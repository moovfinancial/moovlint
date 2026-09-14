package mockcheck_test

import (
	"io"
	"testdata/mockcheck/contracts"
)

type failingRepository struct { // want "test replacement 'failingRepository' implements internal interface 'Repository'"
	contracts.Repository
}

func (*failingRepository) Read() error { return nil }

type embeddedRepository struct{ contracts.Repository }

func (embeddedRepository) Helper() {}

type incidentalHelper struct{}

func (incidentalHelper) Read() error { return nil }

type externalWriter struct{}

func (externalWriter) Write(p []byte) (int, error) { return len(p), nil }

type assignedRepository struct{} // want "test replacement 'assignedRepository' implements internal interface 'Repository'"

func (assignedRepository) Read() error { return nil }

var _ contracts.Repository = assignedRepository{}

type configuredRepository struct{} // want "test replacement 'configuredRepository' implements internal interface 'Repository'"

func (configuredRepository) Read() error { return nil }

var _ = contracts.Dependencies{Repository: configuredRepository{}}

type allowedClient struct{}

func (allowedClient) Transfer() error { return nil }

type returnedRepository struct{} // want "test replacement 'returnedRepository' implements internal interface 'Repository'"

func (returnedRepository) Read() error { return nil }

func returned() contracts.Repository { return returnedRepository{} }

type convertedRepository struct{} // want "test replacement 'convertedRepository' implements internal interface 'Repository'"

func (convertedRepository) Read() error { return nil }

var _ = contracts.Repository(convertedRepository{})

type variadicRepository struct{} // want "test replacement 'variadicRepository' implements internal interface 'Repository'"

func (variadicRepository) Read() error { return nil }

func useMany(...contracts.Repository) {}

func uses() {
	contracts.Use(&failingRepository{})
	contracts.Use(embeddedRepository{})
	contracts.UseClient(allowedClient{})
	useMany(variadicRepository{}, variadicRepository{})
	var writer io.Writer = externalWriter{}
	_, _ = writer.Write(nil)
}
