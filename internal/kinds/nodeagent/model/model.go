package model

import (
	"github.com/caos/orbiter/internal/core/operator"
)

type UserSpec struct {
	operator.NodeAgentSpec `mapstructure:",squash"`
	Verbose                bool
}

type Config struct{}

type Current operator.NodeAgentCurrent
