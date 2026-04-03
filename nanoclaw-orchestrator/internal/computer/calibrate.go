package computer

import (
	"fmt"
	"time"

	"github.com/go-vgo/robotgo"
)

func PrintMousePosition() {
	fmt.Println("Calibration mode: Move mouse to desired positions for 30 seconds...")
	fmt.Println("Press Ctrl+C to stop early")
	fmt.Println("")

	for i := 0; i < 15; i++ {
		x, y := robotgo.GetMousePos()
		fmt.Printf("Mouse at: %d, %d\n", x, y)
		time.Sleep(2 * time.Second)
	}

	fmt.Println("\nCalibration complete!")
	fmt.Println("Use these coordinates for:")
	fmt.Println("  1. PPLX_SEARCH_X, PPLX_SEARCH_Y - Perplexity search box")
	fmt.Println("  2. PPLX_RESPONSE_X, PPLX_RESPONSE_Y - Response copy button area")
}
