package alive

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var artMsg = `
 _______________
< love && peace >
 ---------------
    \
      .-"-.
    _/.-.-.\_
   ( ( o o ) )
    |/  "  \|
     \ .-. /
     /` + "`" + `"""` + "`" + `\
    /       \
`

type Alive struct {
	ArtMsg string `json:"message"`
}

func GetAlive(c *gin.Context) {
	response := Alive{
		ArtMsg: artMsg,
	}

	c.JSON(http.StatusOK, response)
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/alive", GetAlive)
}
