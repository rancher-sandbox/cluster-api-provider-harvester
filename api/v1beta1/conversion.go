package v1beta1

// v1beta1 is the conversion hub (storage version): all other API versions convert to
// and from this one.

// Hub marks HarvesterCluster as a conversion hub.
func (*HarvesterCluster) Hub() {}

// Hub marks HarvesterMachine as a conversion hub.
func (*HarvesterMachine) Hub() {}

// Hub marks HarvesterClusterTemplate as a conversion hub.
func (*HarvesterClusterTemplate) Hub() {}

// Hub marks HarvesterMachineTemplate as a conversion hub.
func (*HarvesterMachineTemplate) Hub() {}
