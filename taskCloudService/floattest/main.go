package main
import "fmt"
func main() {
  f := float64(862031128628060160)
  fmt.Printf("f=%v int64=%d eq=%v\n", f, int64(f), f == float64(int64(f)))
}
