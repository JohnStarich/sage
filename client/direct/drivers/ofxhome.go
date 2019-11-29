//go:generate go run github.com/johnstarich/sage/cmd/ofxhome -ofxhome ../../../out/ofxhome.xml -out generated.go

package drivers

import "github.com/johnstarich/sage/client/direct"

type OFXHomeInstitution struct {
	InstDescription string
	InstFID         string
	InstOrg         string
	InstURL         string
	InstSupport     []direct.DriverMessage
}

func (o OFXHomeInstitution) Description() string {
	return o.InstDescription
}

func (o OFXHomeInstitution) FID() string {
	return o.InstFID
}

func (o OFXHomeInstitution) Org() string {
	return o.InstOrg
}

func (o OFXHomeInstitution) URL() string {
	return o.InstURL
}

func (o OFXHomeInstitution) MessageSupport() []direct.DriverMessage {
	return o.InstSupport
}
