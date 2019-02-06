package v1alpha1

// Auto-generated. DO NOT EDIT.
import (
	"github.com/golang/glog"
	"github.com/xeipuuv/gojsonschema"
)

var introRequestSchema *gojsonschema.Schema

func init() {
	var err error
	introRequestSchema, err = gojsonschema.NewSchema(gojsonschema.NewStringLoader(`{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "properties": {
    "name": {
      "type": "string"
    }
  },
  "type": "object"
}`))
	if err != nil {
		glog.Fatal(err)
	}
}

func (m *IntroRequest) Valid() (*gojsonschema.Result, error) {
	return introRequestSchema.Validate(gojsonschema.NewGoLoader(m))
}
func (m *IntroRequest) IsRequest() {}
