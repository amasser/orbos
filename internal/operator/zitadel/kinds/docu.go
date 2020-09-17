package kinds

import (
	"github.com/caos/orbos/internal/docu"
	"github.com/caos/orbos/internal/operator/zitadel/kinds/iam"
	"github.com/caos/orbos/internal/operator/zitadel/kinds/orb"
)

func GetDocuInfo() []*docu.Type {
	path, orbVersions := orb.GetDocuInfo()

	infos := []*docu.Type{{
		Name: "orb",
		Kinds: []*docu.Info{
			{
				Path:     path,
				Kind:     "orbiter.caos.ch/Orb",
				Versions: orbVersions,
			},
		},
	}}

	infos = append(infos, iam.GetDocuInfo()...)
	return infos
}
