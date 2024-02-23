package app

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/MindHunter86/addie/balancer"
	"github.com/MindHunter86/addie/runtime"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	_ "github.com/MindHunter86/addie/docs"
)

type Controller struct {
	mu sync.RWMutex

	balancers map[balancer.BalancerCluster]balancer.Balancer
	runtime   *runtime.Runtime

	isReady bool
}

func NewController() *Controller {
	return &Controller{}
}

func (m *Controller) SetReady() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.isReady = true
}

func (m *Controller) WithContext(c context.Context) *Controller {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.balancers = c.Value(utils.ContextKeyBalancers).(map[balancer.BalancerCluster]balancer.Balancer)
	m.runtime = c.Value(utils.ContextKeyRuntime).(*runtime.Runtime)
	return m
}

// ---

func respondPlainWithStatus(c *fiber.Ctx, status int) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return c.SendStatus(status)
}

func (m *Controller) getBalancerByString(input string) (_ balancer.BalancerCluster, e error) {
	if input == "" {
		e = fiber.NewError(fiber.StatusNotFound, "cluster could not be empty")
		return
	}

	cluster, ok := balancer.GetBalancerByString[input]
	if !ok {
		e = fiber.NewError(fiber.StatusBadRequest, "invalid cluster name")
		return
	}

	if _, ok := m.balancers[cluster]; !ok {
		panic("internal error, balancer - " + input)
	}

	return cluster, e
}

// @title		Get ConfigStorage Stats
// @version	1.0
// @BasePath	/
func (m *Controller) GetConfigStorageStats(c *fiber.Ctx) error {
	m.runtime.StatsPrint()
	return respondPlainWithStatus(c, fiber.StatusNoContent)
}

// @title			Fiber Example API
// @version		1.0
// @description	This is a sample swagger for Fiber
// @termsOfService	http://swagger.io/terms/
// @contact.name	API Support
// @contact.email	fiber@swagger.io
// @license.name	Apache 2.0
// @license.url	http://www.apache.org/licenses/LICENSE-2.0.html
// @host			localhost:8080
// @BasePath		/
func (m *Controller) GetBalancerStats(c *fiber.Ctx) (e error) {
	cluster, e := m.getBalancerByString(strings.TrimSpace(c.Query("cluster")))
	if e != nil {
		return
	}

	fmt.Fprintln(c, m.balancers[cluster].GetStats())
	return respondPlainWithStatus(c, fiber.StatusOK)
}

// ListAccounts lists all existing accounts
//
//	@Summary		List accounts
//	@Description	get accounts
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Param			q	query		string	false	"name search by q"	Format(email)
//	@Success		204	{array} string
//	@Failure		404	{object}	string
//	@Failure		500	{object}	string
//	@Router			/accounts [get]
func (m *Controller) BalancerStatsReset(c *fiber.Ctx) (e error) {
	cluster, e := m.getBalancerByString(strings.TrimSpace(c.Query("cluster")))
	if e != nil {
		return
	}

	m.balancers[cluster].ResetStats()
	return respondPlainWithStatus(c, fiber.StatusNoContent)
}

func (m *Controller) BalancerUpstreamReset(c *fiber.Ctx) (e error) {
	cluster, e := m.getBalancerByString(strings.TrimSpace(c.Query("cluster")))
	if e != nil {
		return
	}

	m.balancers[cluster].ResetUpstream()
	return respondPlainWithStatus(c, fiber.StatusNoContent)
}

func (m *Controller) BlockIP(c *fiber.Ctx) error {
	ip := strings.TrimSpace(c.Query("ip"))
	if ip == "" {
		return fiber.NewError(fiber.StatusBadRequest, "given ip is empty")
	}

	if net.ParseIP(ip) == nil {
		return fiber.NewError(fiber.StatusBadRequest, "given ip is invalid")
	}

	if e := gConsul.addIpToBlocklist(ip); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	rlog(c).Info().Msgf("ip %s has been banned by %s", ip, c.IP())
	fmt.Fprintln(c, ip+" has been banned")

	return respondPlainWithStatus(c, fiber.StatusOK)
}

func (m *Controller) UnblockIP(c *fiber.Ctx) error {
	ip := strings.TrimSpace(c.Query("ip"))
	if ip == "" {
		return fiber.NewError(fiber.StatusBadRequest, "given ip is empty")
	}

	if net.ParseIP(ip) == nil {
		return fiber.NewError(fiber.StatusBadRequest, "given ip is invalid")
	}

	if e := gConsul.removeIpFromBlocklist(ip); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	rlog(c).Info().Msgf("ip %s has been unbanned by %s", ip, c.IP())
	fmt.Fprintln(c, ip+" has been unbanned")

	return respondPlainWithStatus(c, fiber.StatusOK)
}

func (m *Controller) BlocklistReset(c *fiber.Ctx) error {
	if e := gConsul.resetIpsInBlocklist(); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	return respondPlainWithStatus(c, fiber.StatusNoContent)
}

func (m *Controller) BlocklistSwitch(c *fiber.Ctx) (e error) {
	input := strings.TrimSpace(c.Query("enabled"))
	if input != "0" && input != "1" {
		e = fiber.NewError(fiber.StatusBadRequest, "enabled query can be only 0 or 1")
		return
	}

	if e = gConsul.updateBlocklistSwitcher(input); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	return respondPlainWithStatus(c, fiber.StatusNoContent)
}

func (m *Controller) LimiterSwitch(c *fiber.Ctx) (e error) {
	input := strings.TrimSpace(c.Query("enabled"))
	if input != "0" && input != "1" {
		e = fiber.NewError(fiber.StatusBadRequest, "enabled query can be only 0 or 1")
		return
	}

	if e = gConsul.updateLimiterSwitcher(input); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	return respondPlainWithStatus(c, fiber.StatusNoContent)
}

func (m *Controller) SetLoggerLevel(c *fiber.Ctx) error {
	level := strings.TrimSpace(c.Query("level"))

	switch level {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unknown level sent")
	}

	rlog(c).Error().Msgf("[falsepositive]> new log level applied - %s", level)
	fmt.Fprintln(c, level+" logger level has been applied")

	return respondPlainWithStatus(c, fiber.StatusOK)
}

func (m *Controller) UpdateQualityRewrite(c *fiber.Ctx) (e error) {
	mode, inquality :=
		strings.TrimSpace(c.Query("mode", "soft")),
		strings.TrimSpace(c.Query("level", "1080"))

	if mode != "soft" && mode != "hard" {
		e = fiber.NewError(fiber.StatusInternalServerError, errFbApiInvalidMode.Error())
		return
	}

	quality, ok := utils.GetTitleQualityByString[inquality]
	if !ok {
		e = fiber.NewError(fiber.StatusInternalServerError, errFbApiInvalidQuality.Error())
		return
	}

	gConsul.updateQualityRewrite(quality)

	rlog(c).Info().Msgf("quality %s has been applied by %s", quality.String(), c.IP())
	fmt.Fprintln(c, quality.String()+" has been applied")

	return respondPlainWithStatus(c, fiber.StatusOK)
}
