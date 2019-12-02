// Code generated by goderive DO NOT EDIT.

package operator

// deriveEqualPkg returns whether this and that are equal.
func deriveEqualPkg(this, that *Package) bool {
	return (this == nil && that == nil) ||
		this != nil && that != nil &&
			this.Version == that.Version &&
			deriveEqual(this.Config, that.Config)
}

// deriveEqualSoftware returns whether this and that are equal.
func deriveEqualSoftware(this, that *Software) bool {
	return (this == nil && that == nil) ||
		this != nil && that != nil &&
			deriveEqualPkg(&this.Swap, &that.Swap) &&
			deriveEqualPkg(&this.Kubelet, &that.Kubelet) &&
			deriveEqualPkg(&this.Kubeadm, &that.Kubeadm) &&
			deriveEqualPkg(&this.Kubectl, &that.Kubectl) &&
			deriveEqualPkg(&this.Containerruntime, &that.Containerruntime) &&
			deriveEqualPkg(&this.KeepaliveD, &that.KeepaliveD) &&
			deriveEqualPkg(&this.Nginx, &that.Nginx) &&
			deriveEqualPkg(&this.Hostname, &that.Hostname)
}

// deriveEqualFirewall returns whether this and that are equal.
func deriveEqualFirewall(this, that Firewall) bool {
	if this == nil || that == nil {
		return this == nil && that == nil
	}
	if len(this) != len(that) {
		return false
	}
	for k, v := range this {
		thatv, ok := that[k]
		if !ok {
			return false
		}
		if !(v == thatv) {
			return false
		}
	}
	return true
}

// deriveEqual returns whether this and that are equal.
func deriveEqual(this, that map[string]string) bool {
	if this == nil || that == nil {
		return this == nil && that == nil
	}
	if len(this) != len(that) {
		return false
	}
	for k, v := range this {
		thatv, ok := that[k]
		if !ok {
			return false
		}
		if !(v == thatv) {
			return false
		}
	}
	return true
}