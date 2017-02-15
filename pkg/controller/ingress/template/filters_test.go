package template

import (
	"testing"

	"github.com/flosch/pongo2"
	"github.com/stretchr/testify/assert"
)

func TestHeaderNameFilter(t *testing.T) {
	temp := `
{{ val|header_name }}
{{ val2|header_name }}
	`
	ctx := &pongo2.Context{
		"val":  "hello world",
		"val2": "hello   world",
	}
	res, _ := render(ctx, temp)
	exp := `
hello
hello
	`
	assert.Equal(t, res, exp)
}

func TestHostNameFilter(t *testing.T) {
	temp := `
{{ val|host_name }}
{{ val2|host_name }}
	`
	ctx := &pongo2.Context{
		"val":  "appscode.com",
		"val2": "*.appscode.com",
	}
	res, _ := render(ctx, temp)
	exp := `
hdr(host) -i appscode.com
hdr_end(host) -i .appscode.com
	`
	assert.Equal(t, res, exp)
}

func render(ctx *pongo2.Context, temp string) (string, error) {
	tpl, err := pongo2.FromString(temp)
	if err != nil {
		return "", err
	}
	out, err := tpl.Execute(*ctx)
	if err != nil {
		return "", err
	}
	return out, nil
}
