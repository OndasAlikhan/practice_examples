package client

import (
	"fmt"
	"net/http"
)

type Client struct {
}

func (c Client) Request() error {
	url := "http://localhost:8080"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error get: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("resp: %v\n", resp)

	return nil
}
