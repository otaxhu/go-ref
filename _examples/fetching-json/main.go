package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/otaxhu/go-ref"
)

// dummyjson has up to 100 dummy products to be requested
const urlBase = "https://dummyjson.com/products/"

func main() {
	productId := ref.NewRef(rand.Intn(100) + 1)

	ref.WatchImmediate(func(actualValues, prevValues []int, ctx context.Context) {
		// Passing context to request so it can send a signal when productId Ref has changed and
		// cancel the request
		req, _ := http.NewRequestWithContext(ctx, "GET", urlBase+strconv.Itoa(actualValues[0]), nil)

		// Simulating heavy http request by adding a time.Sleep()
		time.Sleep(500 * time.Millisecond)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer res.Body.Close()

		jsonRes := map[string]any{}

		if err := json.NewDecoder(res.Body).Decode(&jsonRes); err != nil {
			panic(err)
		}

		fmt.Println("Product Title:\n -", jsonRes["title"])
		fmt.Println("Product Description:\n -", jsonRes["description"])
	}, productId)

	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {
		// Try pressing enter many times on the console to see if the requests gets canceled
		productId.SetValue(rand.Intn(100) + 1)
	}
}
