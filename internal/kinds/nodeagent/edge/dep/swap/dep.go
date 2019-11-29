package swap

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	"github.com/caos/infrop/internal/core/operator"
	"github.com/caos/infrop/internal/kinds/nodeagent/adapter"
	"github.com/caos/infrop/internal/kinds/nodeagent/edge/dep"
	"github.com/caos/infrop/internal/kinds/nodeagent/edge/dep/middleware"
)

type Installer interface {
	isSwap()
	adapter.Installer
}

type swapDep struct {
	fstabFilePath string
}

func New(fstabFilePath string) Installer {
	return &swapDep{fstabFilePath}
}

func (swapDep) Is(other adapter.Installer) bool {
	_, ok := middleware.Unwrap(other).(Installer)
	return ok
}

func (swapDep) isSwap() {}

func (swapDep) String() string { return "Swap" }

func (*swapDep) Equals(other adapter.Installer) bool {
	_, ok := other.(*swapDep)
	return ok
}

func (s *swapDep) Current() (pkg operator.Package, err error) {

	var buf bytes.Buffer
	swapon := exec.Command("swapon", "--summary")
	swapon.Stdout = &buf
	if err := swapon.Run(); err != nil {
		return pkg, err
	}

	pkg.Version = "disabled"
	var lines uint8
	var line string
	for {
		if err != nil && err != io.EOF {
			return pkg, err
		}
		line, err = buf.ReadString('\n')
		if len(line) > 0 {
			lines++
		}
		if lines >= 2 {
			pkg.Version = "enabled"
			return
		}
		if err == io.EOF {
			return pkg, nil
		}
	}
}

func (s *swapDep) Ensure(remove operator.Package, ensure operator.Package) (bool, error) {

	return remove.Version != ensure.Version, dep.ManipulateFile(s.fstabFilePath, nil, nil, func(line string) string {
		if !strings.Contains(line, "swap") {
			return line
		}
		switch {
		case strings.HasPrefix(line, "#") && ensure.Version == "enabled" && remove.Version == "disabled":
			return strings.Replace(line, "#", "", 1)
		case !strings.HasPrefix(line, "#") && ensure.Version == "disabled" && remove.Version == "enabled":
			return "#" + line
		}
		return line
	})
}
