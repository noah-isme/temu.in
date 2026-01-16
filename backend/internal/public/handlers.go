package public

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

// RegisterRoutes registers lightweight public endpoints used by the SPA.
// It accepts gin.IRouter so it can be registered against either the root engine
// or a router group (for example, to expose the same handlers under /api).
func RegisterRoutes(rg gin.IRouter) {
    rg.GET("/providers", getProviders)
    rg.GET("/services", getServices)
    rg.GET("/availability", getAvailability)
    rg.POST("/bookings", createBooking)
    rg.POST("/payments/create-intent", createPaymentIntent)
    rg.POST("/payments/confirm", confirmPayment)
}

func getProviders(c *gin.Context) {
    data := []gin.H{
        {"id": "p1", "name": "Provider A", "available": true},
        {"id": "p2", "name": "Provider B", "available": false},
    }
    c.JSON(http.StatusOK, data)
}

func getServices(c *gin.Context) {
    data := []gin.H{
        {"id": "s1", "name": "Haircut", "durationMinutes": 30, "price": 15000},
        {"id": "s2", "name": "Massage", "durationMinutes": 60, "price": 75000},
    }
    c.JSON(http.StatusOK, data)
}

func getAvailability(c *gin.Context) {
    provider := c.Query("providerId")
    date := c.Query("date")
    if date == "" {
        date = time.Now().Format("2006-01-02")
    }
    slots := []string{
        date + "T09:00:00Z",
        date + "T10:00:00Z",
        date + "T14:00:00Z",
    }
    c.JSON(http.StatusOK, gin.H{"providerId": provider, "date": date, "slots": slots})
}

func createBooking(c *gin.Context) {
    var body map[string]interface{}
    if err := c.BindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid"})
        return
    }
    booking := gin.H{
        "id":         "b_" + time.Now().Format("20060102150405"),
        "providerId": body["providerId"],
        "customerName": body["customerName"],
        "slot":       body["slot"],
        "status":     "confirmed",
    }
    c.JSON(http.StatusCreated, booking)
}

func createPaymentIntent(c *gin.Context) {
    var body map[string]interface{}
    _ = c.BindJSON(&body)
    clientSecret := "cs_" + time.Now().Format("150405")
    intent := gin.H{"id": "pi_" + time.Now().Format("20060102150405"), "amount": body["amount"], "currency": body["currency"], "client_secret": clientSecret}
    c.JSON(http.StatusCreated, intent)
}

func confirmPayment(c *gin.Context) {
    var body map[string]interface{}
    _ = c.BindJSON(&body)
    cs, _ := body["client_secret"].(string)
    if cs == "fail" || (cs != "" && string(cs) != cs) {
        c.JSON(http.StatusPaymentRequired, gin.H{"status": "failed", "message": "Payment failed"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "succeeded", "transactionId": "tx_" + time.Now().Format("20060102150405")})
}
