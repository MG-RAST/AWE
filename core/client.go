package core

import (
	"github.com/MG-RAST/AWE/core/uuid"
)

type Client struct {
	Id      string   `bson:"id" json:"id"`
	Name    string   `bson:"name" json:"name"`
	Host    string   `bson:"tasks" json:"tasks"`
	Workers string   `bson:"workers" json:"workers"`
	Cores   int      `bson:"cores" json:"cores"`
	Apps    []string `bson:"apps" json:"apps"`
}

func NewClient() (client *Client) {
	client = new(Client)
	client.Id = uuid.New()
	return
}
