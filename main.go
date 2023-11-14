package main

import (
	"context"
	"fmt"

	j "github.com/samgozman/go-fin-feed/journalist"
)

func main() {
	journalist := j.NewJournalist([]j.NewsProvider{})
	ctx := context.Background()

	fmt.Println(journalist.GetLatestNews(ctx))
}
