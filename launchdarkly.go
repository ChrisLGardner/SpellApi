package main

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	ldclient "gopkg.in/launchdarkly/go-server-sdk.v5"
)

type LaunchDarkly struct {
	*ldclient.LDClient
}

func NewLaunchDarklyClient(key string, timeout int) (*LaunchDarkly, error) {
	ldclient, err := ldclient.MakeClient(key, time.Duration(timeout*int(time.Second)))
	if err != nil {
		return &LaunchDarkly{}, err
	}

	return &LaunchDarkly{ldclient}, nil
}

func (ld *LaunchDarkly) GetUser(ctx context.Context, r *http.Request) lduser.User {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "LaunchDarkly.GetUser")
	defer span.End()

	user := lduser.NewUser(r.Header.Get("X-SPELLAPI-USERID"))

	span.SetAttributes(attribute.Stringer("LaunchDarkly.GetUser.User", user))

	return user
}

func (ld *LaunchDarkly) GetBoolFlag(ctx context.Context, flag string, user lduser.User) bool {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "LaunchDarkly.GetBoolFlag")
	defer span.End()

	span.SetAttributes(attribute.String("LaunchDarkly.GetBoolFlag.Flag", flag))
	span.SetAttributes(attribute.Stringer("LaunchDarkly.GetBoolFlag.User", user))

	res, err := ld.BoolVariation(flag, user, false)
	if err != nil {
		span.SetAttributes(attribute.String("LaunchDarkly.GetBoolFlag.Error", err.Error()))
	}
	span.SetAttributes(attribute.Bool("LaunchDarkly.GetBoolFlag.State", res))

	return res
}

func (ld *LaunchDarkly) GetIntFlag(ctx context.Context, flag string, user lduser.User) int {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "LaunchDarkly.GetIntFlag")
	defer span.End()

	span.SetAttributes(attribute.String("LaunchDarkly.GetIntFlag.Flag", flag))
	span.SetAttributes(attribute.Stringer("LaunchDarkly.GetIntFlag.User", user))

	res, err := ld.IntVariation(flag, user, 0)
	if err != nil {
		span.SetAttributes(attribute.String("LaunchDarkly.GetIntFlag.Error", err.Error()))
	}
	span.SetAttributes(attribute.Int("LaunchDarkly.GetIntFlag.State", res))

	return res
}
