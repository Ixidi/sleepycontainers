package internal

import (
	"html/template"
	"net/http"
)

type Templates struct {
	loading  *template.Template
	shutdown *template.Template
	problem  *template.Template
}

func LoadTemplates() (*Templates, error) {
	loading, err := template.ParseFiles("templates/loading.html")
	if err != nil {
		return nil, err
	}

	shutdown, err := template.ParseFiles("templates/shutdown.html")
	if err != nil {
		return nil, err
	}

	problem, err := template.ParseFiles("templates/problem.html")
	if err != nil {
		return nil, err
	}

	return &Templates{
		loading:  loading,
		shutdown: shutdown,
		problem:  problem,
	}, nil
}

func (t *Templates) WriteLoadingTemplate(w http.ResponseWriter, groupName string) error {
	return t.loading.Execute(w, struct {
		GroupName string
	}{
		GroupName: groupName,
	})
}

func (t *Templates) WriteShutdownTemplate(w http.ResponseWriter, groupName string) error {
	return t.shutdown.Execute(w, struct {
		GroupName string
	}{
		GroupName: groupName,
	})
}

func (t *Templates) WriteProblemTemplate(w http.ResponseWriter, err error) error {
	return t.problem.Execute(w, struct {
		Error error
	}{
		Error: err,
	})
}
