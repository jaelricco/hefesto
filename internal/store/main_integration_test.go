//go:build integration

package store_test

import (
	"os"
	"testing"

	"github.com/jaelricco/hefesto/internal/testutil/pgtest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Run(m)) }
