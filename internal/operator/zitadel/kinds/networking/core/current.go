package core

import (
	"errors"
	"github.com/caos/orbos/internal/operator/zitadel"
	"github.com/caos/orbos/internal/tree"
)

const queriedName = "networking"

type NetworkingCurrent interface {
	GetTlsCertName() string
	GetDomain() string
	GetIssuerSubDomain() string
	GetConsoleSubDomain() string
	GetAPISubDomain() string
	GetAccountsSubDomain() string
	GetReadyCertificate() zitadel.EnsureFunc
}

func ParseQueriedForNetworking(queried map[string]interface{}) (NetworkingCurrent, error) {
	queriedNW, ok := queried[queriedName]
	if !ok {
		return nil, errors.New("no current state for networking found")
	}
	current, ok := queriedNW.(*tree.Tree)
	if !ok {
		return nil, errors.New("current state does not fullfil interface")
	}
	currentNW, ok := current.Parsed.(NetworkingCurrent)
	if !ok {
		return nil, errors.New("current state does not fullfil interface")
	}
	return currentNW, nil
}

func SetQueriedForNetworking(queried map[string]interface{}, networkingCurrent *tree.Tree) {
	queried[queriedName] = networkingCurrent
}
