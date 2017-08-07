package main

import (
	"text/template"
	"github.com/tamalsaha/go-oneliners"
	"os"
)

func main() {
	t1 := template.Must(template.New("n1").Parse(`t1
	`))
	_ = template.Must(t1.New("n2").Parse(`{{ template "n1" }} 123`))

	err := t1.ExecuteTemplate(os.Stdout, "n2", nil)
	if err != nil {
		oneliners.FILE(err)
	}

	err = t1.Execute(os.Stdout, nil)
	if err != nil {
		oneliners.FILE(err)
	}
}
