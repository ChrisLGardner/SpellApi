package main

import (
	"context"
	"net/http"
	"time"

	"github.com/honeycombio/beeline-go"
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

func (ld *LaunchDarkly) GetUser(ctx context.Context, r http.Request) lduser.User {
	ctx, span := beeline.StartSpan(ctx, "LaunchDarkly.GetUser")
	defer span.Send()

	user := lduser.NewUser(r.Header.Get("X-SPELLAPI-USERID"))

	beeline.AddField(ctx, "LaunchDarkly.GetUser.User", user)

	return user
}

func (ld *LaunchDarkly) GetBoolFlag(ctx context.Context, flag string, user lduser.User) bool {
	ctx, span := beeline.StartSpan(ctx, "LaunchDarkly.GetBoolFlag")
	defer span.Send()

	beeline.AddField(ctx, "LaunchDarkly.GetBoolFlag.Flag", flag)
	beeline.AddField(ctx, "LaunchDarkly.GetBoolFlag.User", user)

	res, err := ld.BoolVariation(flag, user, false)
	if err != nil {
		beeline.AddField(ctx, "LaunchDarkly.GetBoolFlag.Error", err)
	}
	beeline.AddField(ctx, "LaunchDarkly.GetBoolFlag.State", res)

	return res
}

func (ld *LaunchDarkly) GetIntFlag(ctx context.Context, flag string, user lduser.User) int {
	ctx, span := beeline.StartSpan(ctx, "LaunchDarkly.GetIntFlag")
	defer span.Send()

	beeline.AddField(ctx, "LaunchDarkly.GetIntFlag.Flag", flag)
	beeline.AddField(ctx, "LaunchDarkly.GetIntFlag.User", user)

	res, err := ld.IntVariation(flag, user, 0)
	if err != nil {
		beeline.AddField(ctx, "LaunchDarkly.GetIntFlag.Error", err)
	}
	beeline.AddField(ctx, "LaunchDarkly.GetIntFlag.State", res)

	return res
}
