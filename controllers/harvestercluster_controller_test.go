/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Extract Server from Kubeconfig", func() {
	var kubeconfig []byte

	BeforeEach(func() {
		kubeconfig = []byte(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJ2akNDQVdPZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQkdNUnd3R2dZRFZRUUtFeE5rZVc1aGJXbGoKYkdsemRHVnVaWEl0YjNKbk1TWXdKQVlEVlFRRERCMWtlVzVoYldsamJHbHpkR1Z1WlhJdFkyRkFNVGN4TXpnMApNVFUwTWpBZUZ3MHlOREEwTWpNd016QTFOREphRncwek5EQTBNakV3TXpBMU5ESmFNRVl4SERBYUJnTlZCQW9UCkUyUjVibUZ0YVdOc2FYTjBaVzVsY2kxdmNtY3hKakFrQmdOVkJBTU1IV1I1Ym1GdGFXTnNhWE4wWlc1bGNpMWoKWVVBeE56RXpPRFF4TlRReU1Ga3dFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRVA3V0RnRnk1NzRWVwp0SVYySzFGMExVZnE1VDJkQlFYVFovUUFIdWVqNDAzMGR1MklvN2tubzZ0SlI5OEJrNVk0bmpDK0VzT3c4UlZvCnJiWkdOVzJJdEtOQ01FQXdEZ1lEVlIwUEFRSC9CQVFEQWdLa01BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0hRWUQKVlIwT0JCWUVGTWJ1c3dyTTZEQS8vcjV2NjNhejJCU3VXSkVjTUFvR0NDcUdTTTQ5QkFNQ0Ewa0FNRVlDSVFETgpKQVhOUHFtZEY4SGViUm5IMTJkTkNVWEY0TXpTd0haSTZwZzVhNDVsd1FJaEFLNHZiRGVjTEIyVzBuQnJ1S0F2ClprNy9lb2JLT05TcEthRzBJdjhHaGhTdQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t
    server: https://10.10.0.10/k8s/clusters/local
  name: local
contexts:
- context:
    cluster: local
    user: local
  name: deathstar
current-context: deathstar
kind: Config
preferences: {}
users:
- name: local
  user:
    token: kubeconfig-user-gqrx7b7k5x:qsp8xzzd4dpb9d99n9fg8vrtdrqndplmlfjtzpkshc9jcndn9fc2ns
`)
	})
	Context("When we provide a kubeconfig with different current-context and cluster.name", func() {
		It("Should still return the right cluster", func() {
			Expect(getHarvesterServerFromKubeconfig(kubeconfig)).To(Equal("https://10.10.0.10/k8s/clusters/local"))
		})
	})
})
