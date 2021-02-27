package wok

import (
	"fmt"
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
			StaticHandler("app.base"),
		},
		"main.error": {
			ControllerHandler(RouteController{
				Controller: "app.error",
				Params:     []string{"errorCode"},
			}),
		},
		"main.user": {
			ControllerHandler(RouteController{
				Controller: "app.user",
				Params:     []string{"userId", "lang"},
			}),
		},
		"main.admin": {
			ControllerPlanHandler(ControllerPlan{
				Controller: "app.user",
				Params:     Params{"userId": {"admin"}},
			}),
		},
	})
}

func TestResolveURL(t *testing.T) {
	parsedURL, _ := url.Parse("/usuarios/asd")
	result := router.ResolveURL(&http.Request{}, parsedURL)
	fmt.Println(result.Controllers)
}

func TestResolveRoute(t *testing.T) {

}
