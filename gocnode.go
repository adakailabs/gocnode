/*
Copyright © 2021 Luis Garcia

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"

	"github.com/CrowdSurge/banner"
	"github.com/adakailabs/gocnode/cmd"
)

// Version Build variable is set at compilation time, to set pass -ldflags "-X main.Build <build sha1>" to go build
var Version string

func main() {
	version()
	cmd.Execute()
}

func version() {
	fmt.Printf("========================================")
	banner.Print("gocnode")
	fmt.Println("========================================")
	fmt.Printf("\nversion:%s\n", Version)
	fmt.Println("========================================")
}
