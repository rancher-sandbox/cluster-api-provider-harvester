# CAPHV : Cluster API Provider Harvester - Guide Quick Start

## Qu'est-ce que Cluster API et pourquoi l'utiliser avec Harvester ?

### Le probleme

Pour creer un cluster Kubernetes sur Harvester, il y a actuellement 3 approches :

| Approche | Description | Limites |
|----------|-------------|---------|
| **Rancher UI** | Clic-clic dans l'interface | Pas reproductible, pas versionnable, pas automatisable |
| **Terraform** | Provider `harvester` + `rancher2` | Pas de reconciliation continue, drift possible, complexite HCL |
| **Cluster API** | Resources Kubernetes declaratives | Ce guide |

### Ce qu'apporte Cluster API (CAPI)

CAPI est un standard Kubernetes qui permet de **gerer des clusters Kubernetes comme des resources Kubernetes**. Concretement :

- Un cluster est un fichier YAML qu'on `kubectl apply`
- La reconciliation est **continue** : si une VM tombe, elle est recreee automatiquement
- Le scaling se fait en changeant `replicas: 3` -> `replicas: 5`
- Les upgrades K8s se font en changeant `version: v1.31.6+rke2r1` -> `version: v1.32.x+rke2r1`
- Tout est versionnable dans Git (GitOps natif)
- L'integration Rancher est automatique via **Rancher Turtles**

### Architecture

```
Rancher Manager (172.16.3.20)           Harvester (172.16.3.100)
+---------------------------------+     +---------------------------+
| CAPI Core Controller            |     |                           |
| RKE2 Bootstrap Provider         |     |  VM capi-test-machine-*   |
| RKE2 Control Plane Provider     |     |  (RKE2 server/agent)      |
| CAPHV Controller         -------|---->|                           |
| Rancher Turtles                 |     |  LB (Harvester LB)        |
+---------------------------------+     +---------------------------+
          |                                         |
          | auto-import via Turtles                 | kubeconfig via LB
          v                                         v
   Rancher UI (visible)                   Workload Cluster API
```

Le **management cluster** (Rancher Manager) heberge tous les controllers CAPI.
Les **workload clusters** sont crees comme VMs sur Harvester.

---

## Pre-requis sur Harvester

Avant de creer des clusters, preparer ces ressources sur Harvester :

### 1. Image VM avec cloud-init

L'image VM doit supporter cloud-init (NoCloud datasource). Images testees :

- `sles15-sp7-minimal-vm.x86_64-cloud-qu2.qcow2` (SLES 15 SP7)

> **Attention** : L'image SLES minimale ne contient pas `iptables`. Les pods
> utilisant `portmap` CNI (comme ingress-nginx) ne demarreront pas. Utiliser
> une image avec `iptables` pre-installe pour la production.

### 2. Keypair SSH

Creer une keypair SSH dans Harvester (UI ou API) :
- Namespace : `default`
- Nom : `capi-ssh-key`

### 3. Reseau VM

Un Network Attachment (NAD) bridge sur le cluster network `mgmt` :
- Namespace : `default`
- Nom : `production`

### 4. Kubeconfig Harvester

Exporter le kubeconfig du cluster Harvester, l'encoder en base64 :

```bash
# Depuis le noeud Harvester
sudo cat /etc/rancher/rke2/rke2.yaml | \
  sed "s|server: https://127.0.0.1:6443|server: https://172.16.3.100:6443|" | \
  base64 -w0
```

### 5. IP Pool pour les VMs (recommande)

Creer un IPPool Harvester pour l'allocation automatique d'IPs :

```bash
# Sur le cluster Harvester
kubectl apply -f - <<EOF
apiVersion: loadbalancer.harvesterhci.io/v1beta1
kind: IPPool
metadata:
  name: capi-vm-pool
spec:
  description: "IP Pool for CAPI VM nodes"
  ranges:
    - subnet: "172.16.0.0/16"
      rangeStart: "172.16.3.40"
      rangeEnd: "172.16.3.49"
      gateway: "172.16.0.1"
EOF
```

> **Point critique** : KubeVirt utilise un bridge binding qui **intercepte le
> trafic DHCP externe**. Les VMs ne peuvent PAS obtenir une IP via DHCP depuis
> le reseau physique. Il faut obligatoirement configurer une **IP statique**
> soit via `vmNetworkConfig` (rc7+, recommande), soit via `networkConfig` manuel.

---

## Installation du provider

### Providers deja installes (via Rancher Turtles)

Sur notre Rancher Manager, Turtles a deja installe :
- CAPI Core v1.10.6
- RKE2 Bootstrap v0.21.1
- RKE2 Control Plane v0.21.1

### Deployer CAPHV

```bash
# Build de l'image (depuis node1)
cd /tmp/caphv
podman build --build-arg TARGETARCH=amd64 . \
  -t gitea.home.zypp.fr/jniedergang/cluster-api-provider-harvester:v0.2.0-rc11

# Transferer l'image sur le management cluster (si pas de registry)
podman save gitea.home.zypp.fr/jniedergang/cluster-api-provider-harvester:v0.2.0-rc11 \
  | ssh rancher@172.16.3.20 'sudo /var/lib/rancher/rke2/bin/ctr \
    --address /run/k3s/containerd/containerd.sock \
    --namespace k8s.io images import -'

# Deployer les CRDs et le controller
# (depuis le management cluster)
kubectl apply -f config/crd/bases/
kubectl label crd harvesterclusters.infrastructure.cluster.x-k8s.io \
  cluster.x-k8s.io/v1beta1=v1alpha1
kubectl label crd harvestermachinetemplates.infrastructure.cluster.x-k8s.io \
  cluster.x-k8s.io/v1beta1=v1alpha1
kubectl label crd harvestermachines.infrastructure.cluster.x-k8s.io \
  cluster.x-k8s.io/v1beta1=v1alpha1
kubectl label crd harvesterclustertemplates.infrastructure.cluster.x-k8s.io \
  cluster.x-k8s.io/v1beta1=v1alpha1

# Deployer le controller manager
kubectl apply -f config/default/
```

---

## Creer un cluster RKE2

### Methode recommandee : avec allocation IP automatique (v0.2.0-rc7+, rc10 recommande)

Cette methode utilise `vmNetworkConfig` sur le HarvesterCluster pour allouer
automatiquement une IP unique a chaque VM depuis un IPPool Harvester.
C'est la seule methode qui supporte le scaling multi-noeud.

```yaml
# capi-cluster.yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: capi-test
  labels:
    # Active l'auto-import dans Rancher via Turtles
    cluster-api.cattle.io/rancher-auto-import: "true"
---
apiVersion: v1
kind: Secret
metadata:
  name: hv-identity-secret
  namespace: capi-test
data:
  # Kubeconfig Harvester encode en base64
  kubeconfig: <BASE64_KUBECONFIG_HARVESTER>
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: capi-test
  namespace: capi-test
  labels:
    ccm: external
    csi: external
spec:
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: RKE2ControlPlane
    name: capi-test-cp
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: HarvesterCluster
    name: capi-test-hv
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: HarvesterCluster
metadata:
  name: capi-test-hv
  namespace: capi-test
spec:
  # Namespace Harvester ou creer les VMs et LBs
  targetNamespace: default
  identitySecret:
    namespace: capi-test
    name: hv-identity-secret
  loadBalancerConfig:
    # "dhcp" pour obtenir une IP automatique pour le LB
    # "pool" pour utiliser un IP Pool Harvester existant
    ipamType: dhcp
    listeners:
      - name: rke2-server
        port: 9345
        protocol: TCP
        backendPort: 9345
  # Allocation IP automatique pour les VMs depuis un IPPool
  vmNetworkConfig:
    ipPoolRef: capi-vm-pool        # Reference a un IPPool Harvester existant
    gateway: "172.16.0.1"
    subnetMask: "255.255.0.0"      # Format string, pas un entier
    dnsServers:
      - "172.16.3.6"
    dnsSearch:
      - "home.lo"
---
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: RKE2ControlPlane
metadata:
  name: capi-test-cp
  namespace: capi-test
spec:
  replicas: 3              # Nombre de control plane nodes
  version: v1.31.6+rke2r1
  rolloutStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
  agentConfig: {}
  serverConfig:
    cni: calico
    cloudProviderName: external
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: HarvesterMachineTemplate
    name: capi-test-machine
    namespace: capi-test
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: HarvesterMachineTemplate
metadata:
  name: capi-test-machine
  namespace: capi-test
spec:
  template:
    spec:
      cpu: 2
      memory: 4Gi
      sshUser: sles
      sshKeyPair: default/capi-ssh-key
      networks:
        - default/production
      # PAS de networkConfig ici -> chaque VM recoit une IP unique du pool
      volumes:
        - volumeType: image
          imageName: default/sles15-sp7-minimal-vm.x86_64-cloud-qu2.qcow2
          volumeSize: 40Gi
          bootOrder: 1
```

### Methode alternative : avec IP statique manuelle

Pour un cluster a 1 noeud, ou si l'on veut controler l'IP exacte :

```yaml
# Dans le HarvesterCluster : PAS de vmNetworkConfig
# Dans le HarvesterMachineTemplate : ajouter networkConfig
spec:
  template:
    spec:
      # ... cpu, memory, etc ...
      networkConfig:
        address: "172.16.3.40/16"
        gateway: "172.16.0.1"
        dnsServers:
          - "172.16.3.6"
        dnsSearch:
          - "home.lo"
```

> **Limite** : avec cette methode, tous les noeuds recoivent la meme IP.
> Utiliser `vmNetworkConfig` (rc7+) pour le multi-noeud.

### Appliquer

```bash
kubectl apply -f capi-cluster.yaml
```

### Suivre la progression

```bash
# Etat global
kubectl get cluster,rke2controlplane,machines,harvestermachines -n capi-test

# IPs allouees
kubectl get harvestermachines -n capi-test \
  -o jsonpath='{range .items[*]}{.metadata.name}: {.status.allocatedIPAddress}{"\n"}{end}'

# Logs du provider
kubectl logs -n caphv-system deploy/caphv-controller-manager -f

# Etat du pool IP
kubectl get ippool.loadbalancer.harvesterhci.io capi-vm-pool -o yaml  # sur Harvester

# Une fois les VMs creees, se connecter en SSH
ssh sles@172.16.3.40

# Verifier RKE2 sur la VM
sudo systemctl status rke2-server
sudo /var/lib/rancher/rke2/bin/kubectl --kubeconfig /etc/rancher/rke2/rke2.yaml get nodes
```

### Initialisation des noeuds (automatique depuis rc10)

Le cloud-provider-harvester a besoin du reseau pod (calico) pour atteindre
l'API Harvester et initialiser les noeuds (supprimer le taint `uninitialized`
et definir le `providerID`). Mais calico est bloque par le taint
`uninitialized` — c'est un probleme oeuf/poule.

**Depuis rc10** : CAPHV resout ce probleme automatiquement. Le controller
set le `providerID` et retire le taint `uninitialized` directement sur le
noeud du workload cluster depuis le management cluster, sans passer par le
cloud-provider. Cela debloque calico, qui debloque le cloud-provider pour
les operations suivantes.

Aucune intervention manuelle n'est requise.

### Import dans Rancher (automatique)

L'import est **entierement automatique** grace a Rancher Turtles :

1. Le label `cluster-api.cattle.io/rancher-auto-import: "true"` sur le namespace
   fait que **Turtles cree automatiquement** le cluster dans Rancher
2. Turtles deploie automatiquement le `cattle-cluster-agent` dans le workload cluster
3. L'agent s'enregistre aupres de Rancher, le cluster apparait dans l'UI

Verifie sur le cluster `capi-test` : importe automatiquement comme `c-kzz9c` dans
Rancher, statut Active, sans aucune intervention manuelle.

### Supprimer un cluster

```bash
kubectl delete cluster capi-test -n capi-test
```

Cela supprime automatiquement : les VMs, les PVCs, le Load Balancer, les
IP Pools crees par le controller, et les secrets cloud-init sur Harvester.
Les IPs sont liberees dans le pool.

---

## Scaling

### Scaler le control plane

```bash
kubectl patch rke2controlplane capi-test-cp -n capi-test \
  --type=merge -p '{"spec":{"replicas":5}}'
```

Chaque nouveau noeud recoit automatiquement une IP unique depuis le pool.
Les IPs sont liberees quand les machines sont supprimees.

Le RKE2ControlPlane cree les machines une par une : il attend que chaque
noeud soit Ready et que les conditions `AgentHealthy` et `EtcdMemberHealthy`
soient satisfaites avant de creer le suivant.

### Scale down

```bash
kubectl patch rke2controlplane capi-test-cp -n capi-test \
  --type=merge -p '{"spec":{"replicas":1}}'
```

Les machines sont supprimees progressivement. Les IPs sont liberees dans le pool.

---

## Resultat teste (v0.2.0-rc11, 2026-03-06)

Cluster `capi-test` avec 3 control plane nodes :

| Noeud | IP | Statut | Roles |
|-------|----|--------|-------|
| `capi-test-machine-hw24q` | 172.16.3.42 | Ready | control-plane,etcd,master |
| `capi-test-machine-lzjrd` | 172.16.3.43 | Ready | control-plane,etcd,master |
| `capi-test-machine-2jjjk` | 172.16.3.47 | Ready | control-plane,etcd,master |

IPPool `capi-vm-pool` : 3/10 allouees, 7 disponibles.
Cluster importe dans Rancher : `c-kzz9c`, statut Active.

### Test de remediation automatique (rc10)

Scenario : suppression d'une VM control-plane → remplacement sans intervention manuelle.

Timeline :
- T+0 : VM `capi-test-machine-b9sm6` (172.16.3.46) supprimee sur Harvester
- T+0s : Noeud passe `NotReady`, MHC detecte le noeud unhealthy
- T+5min : MHC depasse le seuil (5min NotReady), marque la Machine pour suppression
- T+5min : ReconcileDelete : IP liberee, nettoyage etcd execute
  (membre deja retire par RKE2ControlPlane → "nothing to remove")
- T+5min : Secret cloud-init et VM supprimes, nouvelle Machine cree par RKE2ControlPlane
- T+6min : Nouvelle VM `capi-test-machine-2jjjk` Running sur Harvester (IP 172.16.3.47)
- T+8min : Nouveau noeud rejoint le cluster, providerID et taint geres automatiquement
- T+9min : 3 noeuds `Ready`, cluster pleinement operationnel

Aucune intervention manuelle requise — ni pour l'initialisation, ni pour le remplacement.

---

## Limites connues

### Limites critiques (impactent l'utilisation en production)

| Limite | Impact | Contournement |
|--------|--------|---------------|
| **Pas de DHCP** a travers le bridge KubeVirt | Chaque VM necessite une IP statique explicite | `vmNetworkConfig` (rc7+) ou `networkConfig` manuel |
| ~~Pas de scaling automatique des IPs~~ | **Resolu en rc7** | Utiliser `vmNetworkConfig` avec un IPPool Harvester |
| ~~Cloud provider : bootstrap du premier noeud~~ | **Resolu en rc10** | CAPHV initialise les noeuds depuis le management cluster |
| **Image SLES minimale sans iptables** | ingress-nginx et les pods avec portmap CNI ne demarrent pas | Utiliser une image avec iptables ou l'installer via cloud-init runcmd |
| ~~Import Rancher pas entierement automatique~~ | **Automatique via Turtles** | Fonctionne sans intervention depuis rc9+ |

### Limites du provider (alpha, v0.2.0-rc11)

| Limite | Detail |
|--------|--------|
| ~~Pas de webhook de validation~~ | **Disponible en rc10** avec `--enable-webhooks` (necessite cert-manager) |
| **Version alpha** | API `v1alpha1`, peut changer sans backward compatibility |

### Bugs corriges dans rc7

| Bug | Impact | Correction |
|-----|--------|------------|
| `Store.Reserve()` ne mettait pas a jour `Status.Allocated` | Toutes les machines recevaient la meme IP (premiere du pool) | Ajout de l'ecriture dans `Allocated` + decrementation `Available` |
| Regex `CheckNamespacedName` sans underscore | Les noms d'images contenant `x86_64` etaient rejetes | Ajout de `_` dans la classe de caracteres regex |

### Bug corrige dans rc8

| Bug | Impact | Correction |
|-----|--------|------------|
| Topologie CPU : `sockets * cores * threads` applique 3 fois | VMs recevaient CPU^3 vCPUs (ex: 2 CPU → 8 vCPUs) | `sockets=1, threads=1, cores=N` pour N vCPUs demandes |

### Amelioration rc9 : nettoyage etcd automatique

| Feature | Detail |
|---------|--------|
| **Nettoyage etcd automatique** | A la suppression d'un noeud control-plane, CAPHV retire le membre etcd du workload cluster via `kubectl exec etcdctl` avant de supprimer la VM |
| **Non-bloquant** | Toute erreur est loguee en warning, la suppression VM continue toujours |
| **Filet de securite** | RKE2ControlPlane (CAPI v1.10+) gere deja le retrait etcd dans la plupart des cas ; ce code couvre les cas limites (cluster injoignable, timeout, etc.) |
| **Fichiers** | `util/etcd.go` (helpers), `util/etcd_test.go` (11 tests), 1 methode + 1 ligne dans `harvestermachine_controller.go` |

### Ameliorations rc10 : node init + webhooks + import Rancher automatique

| Feature | Detail |
|---------|--------|
| **Initialisation noeud depuis le management cluster** | CAPHV set le `providerID` et retire le taint `uninitialized` directement via la kubeconfig du workload cluster, sans passer par le cloud-provider. Resout le probleme oeuf/poule calico↔cloud-provider |
| **Webhooks de validation** | `admission.CustomValidator` pour HarvesterMachine (cpu, memory, sshUser, sshKeyPair, volumes, networks, networkConfig) et HarvesterCluster (targetNamespace, identitySecret, ipamType, vmNetworkConfig). Actives via `--enable-webhooks` |
| **Import Rancher confirme automatique** | Rancher Turtles deploie cattle-cluster-agent automatiquement. Verifie : cluster `capi-test` importe comme `c-kzz9c` sans intervention |
| **Fichiers** | `util/node_init.go` + tests, `api/v1alpha1/*_webhook.go`, `cmd/main.go` modifie |

### Corrections rc11 : PVCs orphelins + compatibilite KubeVirt memory.guest

| Fix | Detail |
|-----|--------|
| **PVCs orphelins a la suppression VM** | L'ancien flux supprimait les PVCs pendant que la VM terminait → webhook Harvester bloquait → au retry, VM disparue → PVCs jamais nettoyes. Nouveau flux : supprimer VM → requeue 10s → lister et supprimer PVCs par prefixe une fois la VM partie |
| **`memory.guest` manquant dans le domain spec** | Apres upgrade Harvester/KubeVirt, `memory.guest` ou `resources.limits.memory` est obligatoire. Les VMs creees avec seulement `resources.requests.memory` ne demarraient plus et n'affichaient pas la memoire dans l'UI. Corrige : `memory.guest` est maintenant defini sur chaque VM creee |
| **Procedure de redemarrage post-upgrade Harvester** | Apres upgrade, les VMs existantes necessitent un patch manuel : `kubectl patch vm <name> --type=json -p '[{"op":"add","path":"/spec/template/spec/domain/memory","value":{"guest":"<MEM>"}}]'`. De plus, `spec.running` est deprecie → utiliser `spec.runStrategy: Always` |

### Limites vs Terraform

| | CAPI/CAPHV | Terraform |
|---|---|---|
| **Reconciliation continue** | Oui (controller loop) | Non (run manuel) |
| **Multi-disk** | Non (1 disk) | Oui |
| **DHCP** | Non (bridge KubeVirt) | Oui (cloud-init gere par Harvester) |
| **Maturite** | Alpha | Stable |
| **Scaling** | Declaratif (replicas) avec IP pool auto | Manuel |
| **Upgrade K8s** | Changement de version dans le YAML | Recreer les VMs |
| **GitOps** | Natif (resources K8s) | Possible (tfstate) |
| **Rancher integration** | Automatique via Turtles | Via provider rancher2 |

### Limites vs Rancher UI (provisioning natif)

| | CAPI/CAPHV | Rancher native |
|---|---|---|
| **Simplicite** | YAML complexe (6 objets) | Interface graphique |
| **Reproductibilite** | Excellente (versionnable) | Faible |
| **Harvester support** | Bridge networks seulement | Complet (masquerade, DHCP...) |
| **Cloud provider** | Automatique (rc10+) | Automatique |
| **Import Rancher** | Automatique (Turtles) | Natif |

---

## Conclusion

CAPI avec CAPHV est interessant pour :
- **L'approche GitOps** : les clusters sont des fichiers YAML versionnables
- **La reconciliation continue** : si une VM tombe, elle est recreee
- **La standardisation** : meme workflow que AWS, Azure, vSphere via CAPI
- **Le scaling declaratif** : changer `replicas:` suffit, les IPs sont gerees automatiquement

Depuis v0.2.0-rc7, le scaling multi-noeud fonctionne grace a l'allocation
IP automatique depuis les IP Pools Harvester. Le provider reste en alpha
(`v1alpha1`) mais couvre les cas d'usage essentiels : creation declarative,
scaling, reconciliation continue, et integration Rancher via Turtles.

Depuis rc8, la topologie CPU est correcte (sockets=1, threads=1, cores=N).

Depuis rc9, le nettoyage etcd est automatique lors de la suppression de
noeuds control-plane.

Depuis rc10, l'initialisation des noeuds est entierement automatique (plus
besoin d'intervention manuelle sur le premier noeud), l'import Rancher est
confirme automatique via Turtles, et des webhooks de validation sont
disponibles. Le cycle complet fonctionne sans intervention :
creation → scaling → remediation → remplacement en ~9 minutes.

Depuis rc11, les PVCs orphelins sont correctement nettoyes a la suppression
de VMs, et la compatibilite avec les nouvelles versions de KubeVirt
(memory.guest obligatoire, runStrategy au lieu de running) est assuree.

La seule limitation operationnelle restante est le besoin d'une image VM
avec `iptables` pour les pods utilisant portmap CNI.

Pour un usage production sur Harvester aujourd'hui, Terraform ou le
provisioning natif Rancher restent plus matures, mais CAPHV offre une
approche GitOps superieure pour les environnements multi-clusters.
