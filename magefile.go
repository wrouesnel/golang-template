//go:build mage

// The project's build: golang-template's (magebuild), as its targets, with
// what differs here in Configure.
package main

import (
	//mage:import
	"github.com/wrouesnel/golang-template/magebuild"
)

func init() {
	magebuild.Configure(magebuild.Config{})
}
