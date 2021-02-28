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
			ControllerHandler(RouteController{
				Controller: "app.landing",
				Params:     []string{"foo"},
			}),
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

func TestResolveURL(t *testing.T) {

	type testCase struct {
		url            string
		expectedResult RouteResult
	}

	testCases := []testCase{
		{
			url: "/?foo=bar&foo=baz",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.landing", Params: Params{"foo": {"bar", "baz"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			url: "/usuarios/pepe",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.user", Params: Params{"userId": {"pepe"}, "lang": {"es"}, "dialect": {"ES"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			url: "/users/pepe",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.user", Params: Params{"userId": {"pepe"}, "lang": {"en"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			url: "/users/john",
			expectedResult: RouteResult{
				ReloadOn: []string{"newUser"},
				Controllers: []ControllerPlan{
					{Controller: "app.error", Params: Params{"errorCode": {"404"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			url: "/users/admin",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.user", Params: Params{"userId": {"adminSuperPlus"}}},
					{Controller: "app.base"},
				},
			},
		},
	}

	for _, testCase := range testCases {
		parsedURL, _ := url.Parse(testCase.url)
		result := router.ResolveURL(&http.Request{}, parsedURL)

		jsonResult, _ := json.Marshal(result)
		expectedJSON, _ := json.Marshal(testCase.expectedResult)

		if string(jsonResult) != string(expectedJSON) {
			t.Error("\n\nGot:\n\n" + string(jsonResult) + "\n\nExpected:\n\n" + string(expectedJSON))
		}
	}
}

func TestResolveRoute(t *testing.T) {

}
