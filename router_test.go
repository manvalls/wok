package wok

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

var router *LocalRouter = NewLocalRouter()

func init() {
	router.AddRoutes(Routes{
		"main.landing": RoutePath("/"),
		"main.user": RoutePaths{
			"/usuarios/:userId": ExtraParams{
				"lang":    "es",
				"dialect": "ES",
			},
			"/users/:userId": ExtraParams{
				"lang": "en",
			},
		},
		"main.admin": RoutePaths{
			"/usuarios/admin": ExtraParams{
				"lang":    "es",
				"dialect": "ES",
			},
			"/users/admin": ExtraParams{
				"lang": "en",
			},
		},
		"main.error": RoutePaths{
			"/:page*": ExtraParams{
				"errorCode": "404",
			},
		},
	})

	router.Map("main", func(r *http.Request, route string, params Params) RouteRedirection {
		return RouteRedirection{
			Route:  route,
			Params: params,
		}
	})

	router.Map("main.user", func(r *http.Request, route string, params Params) RouteRedirection {
		if params.Get("userId") == "john" {
			return RouteRedirection{
				Route: "main.error",
				Params: Params{
					"errorCode": {"404"},
					"page":      params["page"],
				},
				ReloadOn: []string{"newUser"},
			}
		}

		return RouteRedirection{
			Route:  route,
			Params: params,
		}
	})

	router.HandleRoutes(RouteHandlers{
		"main": {
			StaticHandler("app.base"),
		},
		"main.landing": {
			StaticHandler("app.landing"),
		},
		"main.error": {
			ControllerHandler(RouteController{
				Controller: "app.error",
				Params:     []string{"errorCode", "page"},
			}),
		},
		"main.user": {
			ControllerHandler(RouteController{
				Controller: "app.user",
				Params:     []string{"userId", "lang", "dialect"},
			}),
		},
		"main.admin": {
			ControllerPlanHandler(ControllerPlan{
				Controller: "app.user",
				Params:     Params{"userId": {"adminSuperPlus"}},
			}),
		},
	})
}

func TestResolveLanding(t *testing.T) {
	parsedURL, _ := url.Parse("/")
	result := router.ResolveURL(&http.Request{}, parsedURL)

	jsonResult, _ := json.Marshal(result)
	expectedJSON, _ := json.Marshal(RouteResult{
		Controllers: []ControllerPlan{
			{Controller: "app.landing"},
			{Controller: "app.base"},
		},
	})

	if string(jsonResult) != string(expectedJSON) {
		t.Error("Got" + string(jsonResult) + ", expected " + string(expectedJSON))
	}
}

func TestResolveRoute(t *testing.T) {

}
