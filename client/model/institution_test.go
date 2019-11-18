package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrg(t *testing.T) {
	someOrg := "some org"
	assert.Equal(t, someOrg, BasicInstitution{InstOrg: someOrg}.Org())
}

func TestFID(t *testing.T) {
	someFID := "some fid"
	assert.Equal(t, someFID, BasicInstitution{InstFID: someFID}.FID())
}

func TestDescription(t *testing.T) {
	someDescription := "some description"
	assert.Equal(t, someDescription, BasicInstitution{InstDescription: someDescription}.Description())
}
