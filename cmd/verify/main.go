package main

import (
	"fmt"
	"image/png"
	"os"

	gosvg "github.com/Blue-Mink/gosvg"
)

func main() {
	for _, name := range []string{"cn", "s60s", "lucide"} {
		data, err := os.ReadFile("testdata/" + name + ".svg")
		if err != nil {
			fmt.Println("READ ERR", name, err)
			continue
		}
		img, err := gosvg.Render(data, 256)
		if err != nil {
			fmt.Println("RENDER ERR", name, err)
			continue
		}
		out, _ := os.Create("out_" + name + ".png")
		png.Encode(out, img)
		out.Close()
		fmt.Println("OK", name)
	}
}
