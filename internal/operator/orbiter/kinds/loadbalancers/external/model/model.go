package model

import "github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/core/infra"

type UserSpec infra.Address

type Config struct{}

var CurrentVersion = "v0"

type Current struct {
	Address infra.Address
}
