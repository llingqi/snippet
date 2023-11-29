package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"moliang.net/snippetbox/ui"
	"net/http"
)

// Update the signature for the routes() method so that it returns a
// http.Handler instead of *http.ServeMux.
/*func (app *application) routes() http.Handler {
	//创建路由，注册home作为"/"的handle,serveMux对待"/"像是捕捉全部请求
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("./ui/static/"))         //从目录下查找文件
	mux.Handle("/static/", http.StripPrefix("/static", fileServer)) //去除url前缀/static

	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/snippet/view", app.snippetView)
	mux.HandleFunc("/snippet/create", app.snippetCreate)

	mux.HandleFunc("/static/file.zip", app.downloadHandler) //怎么清理路径？不用清理，不知道为啥。访问路径http://localhost:4000/static/file.zip
	//// Pass the servemux as the 'next' parameter to the secureHeaders middleware.
	//// Because secureHeaders is just a function, and the function returns a
	//// http.Handler we don't need to do anything else.
	//// Wrap the existing chain with the logRequest middleware.
	//return app.recoverPanic(app.logRequest(secureHeaders(mux)))

	// Create a middleware chain containing our 'standard' middleware
	// which will be used for every request our application receives.
	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	// Return the 'standard' middleware chain followed by the servemux.
	return standard.Then(mux)
	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/snippet/view/:id", app.snippetView)
}*/
func (app *application) routes() http.Handler {
	// Initialize the router.
	router := httprouter.New()
	// Create a handler function which wraps our notFound() helper, and then
	// assign it as the custom handler for 404 Not Found responses. You can also
	// set a custom handler for 405 Method Not Allowed responses by setting
	// router.MethodNotAllowed in the same way too.
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	// Take the ui.Files embedded filesystem and convert it to a http.FS type so
	// that it satisfies the http.FileSystem interface. We then pass that to the
	// http.FileServer() function to create the file server handler.
	fileServer := http.FileServer(http.FS(ui.Files))

	// Our static files are contained in the "static" folder of the ui.Files
	// embedded filesystem. So, for example, our CSS stylesheet is located at
	// "static/css/main.css". This means that we now longer need to strip the
	// prefix from the request URL -- any requests that start with /static/ can
	// just be passed directly to the file server and the corresponding static
	// file will be served (so long as it exists).
	//css等静态文件放在static字段下，不需要去掉，然后再访问了
	router.Handler(http.MethodGet, "/static/*filepath", fileServer)

	router.HandlerFunc(http.MethodGet, "/ping", ping)
	//// Update the pattern for the route for the static files.
	//fileServer := http.FileServer(http.Dir("./ui/static/"))
	//router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

	// Create a new middleware chain containing the middleware specific to our
	// dynamic application routes. For now, this chain will only contain the
	// LoadAndSave session middleware but we'll add more to it later.
	// Then chains the middleware and returns the final http.Handler.
	//     New(m1, m2, m3).Then(h)
	// is equivalent to:
	//     m1(m2(m3(h)))
	// Use the nosurf middleware on all our 'dynamic' routes.
	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	// And then create the routes using the appropriate methods, patterns and
	// handlers.
	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/snippet/view/:id", dynamic.ThenFunc(app.snippetView))

	// Add the five new routes, all of which use our 'dynamic' middleware chain.
	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))
	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))

	router.Handler(http.MethodGet, "/about", dynamic.ThenFunc(app.about))
	router.Handler(http.MethodGet, "/account/view", dynamic.ThenFunc(app.accountView))

	// 添加路由更新用户密码
	router.Handler(http.MethodGet, "/account/password/update", dynamic.ThenFunc(app.accountPasswordUpdate))
	router.Handler(http.MethodPost, "/account/password/update", dynamic.ThenFunc(app.accountPasswordUpdatePost))

	// Protected (authenticated-only) application routes, using a new "protected"
	// middleware chain which includes the requireAuthentication middleware.
	// Because the 'protected' middleware chain appends to the 'dynamic' chain
	// the noSurf middleware will also be used on the three routes below too.
	protected := dynamic.Append(app.requireAuthentication)

	router.Handler(http.MethodGet, "/snippet/create", protected.ThenFunc(app.snippetCreate))
	router.Handler(http.MethodPost, "/snippet/create", protected.ThenFunc(app.snippetCreatePost))
	router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogoutPost))
	// Create the middleware chain as normal.
	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	// Wrap the router with the middleware and return it as normal.
	return standard.Then(router)
}
