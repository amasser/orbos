package orbiter

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
)

type Orb struct {
	URL       string
	Repokey   string
	Masterkey string
}

var (
	ipPartRegex = `([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])`
	ipRegex     = fmt.Sprintf(`%s\.%s\.%s\.%s`, ipPartRegex, ipPartRegex, ipPartRegex, ipPartRegex)
	cidrRegex   = fmt.Sprintf(`%s/([1-2][0-9]|3[0-2]|[0-9])`, ipRegex)

	compiledIP   = regexp.MustCompile(fmt.Sprintf(`^(%s)$`, ipRegex))
	compiledCIDR = regexp.MustCompile(fmt.Sprintf(`^(%s)$`, cidrRegex))
)

type IPAddress string

type CIDR string

func (c CIDR) Validate() error {
	if !compiledCIDR.MatchString(string(c)) {
		return errors.Errorf("Value %s is not in valid CIDR notation. It does not match the regular expression %s", c, compiledCIDR.String())
	}
	return nil
}

func (i IPAddress) Validate() error {
	if !compiledIP.MatchString(string(i)) {
		return errors.Errorf("Value %s is not a valid IP address. It does not match the regular expression %s", i, compiledIP.String())
	}
	return nil
}
