package app

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/MindHunter86/addie/internal/balancer"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	mu        sync.RWMutex
	balancers map[balancer.BalancerCluster]balancer.Balancer

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

func (m *Controller) WithContext(_ context.Context, bare, cloud balancer.Balancer) *Controller {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.balancers = map[balancer.BalancerCluster]balancer.Balancer{
		balancer.BalancerClusterCloud: cloud,
		balancer.BalancerClusterNodes: bare,
	}

	return m
}

func (m *Controller) GetBalancerStats(c *fiber.Ctx) (e error) {
	cluster, e := m.getBalancerByString(strings.TrimSpace(c.Query("cluster")))
	if e != nil {
		return
	}

	fmt.Fprintln(c, m.balancers[cluster].GetStats())
	return respondPlainWithStatus(c, fiber.StatusOK)
}

func (m *Controller) BalancerStatsReset(c *fiber.Ctx) (e error) {
	cluster, e := m.getBalancerByString(strings.TrimSpace(c.Query("cluster")))
	if e != nil {
		return
	}

	m.balancers[cluster].ResetStats()
	return respondPlainWithStatus(c, fiber.StatusNoContent)
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

// TODO - waiting migration on Dynamic Config
// func (m *Controller) UpdateQualityRewrite(c *fiber.Ctx) (e error) {
// 	mode, inquality :=
// 		strings.TrimSpace(c.Query("mode", "soft")),
// 		strings.TrimSpace(c.Query("level", "1080"))

// 	if mode != "soft" && mode != "hard" {
// 		e = fiber.NewError(fiber.StatusInternalServerError, errFbApiInvalidMode.Error())
// 		return
// 	}

// 	quality, ok := utils.GetTitleQualityByString[inquality]
// 	if !ok {
// 		e = fiber.NewError(fiber.StatusInternalServerError, errFbApiInvalidQuality.Error())
// 		return
// 	}

// 	// gConsul.updateQualityRewrite(quality)

// 	rlog(c).Info().Msgf("quality %s has been applied by %s", quality.String(), c.IP())
// 	fmt.Fprintln(c, quality.String()+" has been applied")

// 	return respondPlainWithStatus(c, fiber.StatusOK)
// }
