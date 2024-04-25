package backend

import (
    "context"
    "fmt"

    "appstore/constants"

    "github.com/olivere/elastic/v7"
)

var (
    ESBackend *ElasticsearchBackend
)

type ElasticsearchBackend struct {
    client *elastic.Client
}
