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
			"/users/admin": ExtraParams{},
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
			url: "/foo/bar",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.error", Params: Params{"errorCode": {"404"}, "page": {"foo/bar"}}},
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
	type testCase struct {
		route          string
		params         Params
		expectedURL    string
		expectedResult RouteResult
	}

	testCases := []testCase{
		{
			route:       "main.landing",
			params:      Params{"foo": {"bar", "baz"}},
			expectedURL: "/?foo=bar&foo=baz",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.landing", Params: Params{"foo": {"bar", "baz"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			route:       "main.user",
			params:      Params{"userId": {"pepe"}, "lang": {"es"}, "dialect": {"ES"}},
			expectedURL: "/usuarios/pepe",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.user", Params: Params{"userId": {"pepe"}, "lang": {"es"}, "dialect": {"ES"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			route:       "main.user",
			params:      Params{"userId": {"pepe"}, "lang": {"en"}},
			expectedURL: "/users/pepe",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.user", Params: Params{"userId": {"pepe"}, "lang": {"en"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			route:       "main.admin",
			expectedURL: "/users/admin",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.user", Params: Params{"userId": {"adminSuperPlus"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			route:       "main.admin",
			params:      Params{"lang": {"es"}, "dialect": {"ES"}},
			expectedURL: "/usuarios/admin",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.user", Params: Params{"userId": {"adminSuperPlus"}}},
					{Controller: "app.base"},
				},
			},
		},
		{
			route:       "main.error",
			params:      Params{"errorCode": {"404"}, "page": {"foo/bar"}},
			expectedURL: "/foo/bar",
			expectedResult: RouteResult{
				Controllers: []ControllerPlan{
					{Controller: "app.error", Params: Params{"errorCode": {"404"}, "page": {"foo/bar"}}},
					{Controller: "app.base"},
				},
			},
		},
	}

	for _, testCase := range testCases {
		url := router.ResolveRoute(&http.Request{}, testCase.route, testCase.params)

		if url != testCase.expectedURL {
			t.Error("\n\nGot:\n\n" + url + "\n\nExpected:\n\n" + testCase.expectedURL)
		}
	}
}
