package alive

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var monkeySayings = []string{
	"love && peace",
	"bananas are tasty",
	"go bananas!",
	"ooh ooh ah ah",
	"code more, sleep less",
	"monkey see, monkey do",
	"throw no banana",
	"climb every mountain",
	"eat sleep code repeat",
}

type Alive struct {
	ArtMsg string `json:"message"`
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

func generateMonkeyArt(saying string) string {
	// Calculate the length of the saying
	lineLength := len(saying) + 2 // 2 spaces

	// Generate the top and bottom lines
	topLine := " " + strings.Repeat("_", lineLength)
	bottomLine := " " + strings.Repeat("-", lineLength)

	// Generate the monkey art with the saying
	monkeyArt := fmt.Sprintf(`
%s
< %s >
%s
    \
      .-"-.
    _/.-.-.\_
   ( ( o o ) )
    |/  "  \|
     \ .-. /
     /`+"`"+`"""`+"`"+`\
    /       \
`, topLine, saying, bottomLine)

	return monkeyArt
}

func GetAlive(c *gin.Context) {
	saying := monkeySayings[rand.Intn(len(monkeySayings))]

	response := Alive{
		ArtMsg: generateMonkeyArt(saying),
	}

	c.JSON(http.StatusOK, response)
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/alive", GetAlive)
}
