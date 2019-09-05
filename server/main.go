package main

import (
	"context"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/graph-gophers/graphql-transport-ws/graphqlws"
)

const schema = `
	schema {
		subscription: Subscription
		mutation: Mutation
		query: Query
	}

	type Query {
		hello: String!
	}

	type Subscription {
		helloSaid(): HelloSaidEvent!
	}

	type Mutation {
		sayHello(msg: String!): HelloSaidEvent!
	}

	type HelloSaidEvent {
		id: String!
		msg: String!
	}
`

var httpPort = 4000

func init() {
	port := os.Getenv("HTTP_PORT")
	if port != "" {
		var err error
		httpPort, err = strconv.Atoi(port)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	// graphiql handler
	http.HandleFunc("/", http.HandlerFunc(graphiql))

	// init graphQL schema
	s, err := graphql.ParseSchema(schema, newResolver())
	if err != nil {
		panic(err)
	}

	// graphQL handler
	graphQLHandler := graphqlws.NewHandlerFunc(s, &relay.Handler{Schema: s})
	http.HandleFunc("/graphql", graphQLHandler)

	// start HTTP server
	if err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil); err != nil {
		panic(err)
	}
}

type resolver struct {
	helloSaidEvents     chan *helloSaidEvent
	helloSaidSubscriber chan *helloSaidSubscriber
}

func newResolver() *resolver {
	r := &resolver{
		helloSaidEvents:     make(chan *helloSaidEvent),
		helloSaidSubscriber: make(chan *helloSaidSubscriber),
	}

	go r.broadcastHelloSaid()

	return r
}

func (r *resolver) Hello() string {
	return "Hello world!"
}

func (r *resolver) SayHello(args struct{ Msg string }) *helloSaidEvent {
	e := &helloSaidEvent{msg: args.Msg, id: randomID()}
	go func() {
		select {
		case r.helloSaidEvents <- e:
		case <-time.After(1 * time.Second):
		}
	}()
	return e
}

type helloSaidSubscriber struct {
	stop   <-chan struct{}
	events chan<- *helloSaidEvent
}

func (r *resolver) broadcastHelloSaid() {
	subscribers := map[string]*helloSaidSubscriber{}
	unsubscribe := make(chan string)

	// NOTE: subscribing and sending events are at odds.
	for {
		select {
		case id := <-unsubscribe:
			delete(subscribers, id)
		case s := <-r.helloSaidSubscriber:
			subscribers[randomID()] = s
		case e := <-r.helloSaidEvents:
			for id, s := range subscribers {
				go func(id string, s *helloSaidSubscriber) {
					select {
					case <-s.stop:
						unsubscribe <- id
						return
					default:
					}

					select {
					case <-s.stop:
						unsubscribe <- id
					case s.events <- e:
					case <-time.After(time.Second):
					}
				}(id, s)
			}
		}
	}
}

func (r *resolver) HelloSaid(ctx context.Context) <-chan *helloSaidEvent {
	c := make(chan *helloSaidEvent)
	// NOTE: this could take a while
	r.helloSaidSubscriber <- &helloSaidSubscriber{events: c, stop: ctx.Done()}

	go func() {
		select {
		case <-time.After(3 * time.Second):
			close(c)
			return
		}
	}()

	return c
}

type helloSaidEvent struct {
	id  string
	msg string
}

func (r *helloSaidEvent) Msg() string {
	return r.msg
}

func (r *helloSaidEvent) ID() string {
	return r.id
}

func randomID() string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, 16)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

var graphiql = func(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("graphiql").Parse(`
<!DOCTYPE html>
<html>

<head>
  <meta charset=utf-8/>
  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>

<body>
  <div id="root">
    <style>
      body {
        background-color: rgb(23, 42, 58);
        font-family: Open Sans, sans-serif;
        height: 90vh;
      }
      #root {
        height: 100%;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .loading {
        font-size: 32px;
        font-weight: 200;
        color: rgba(255, 255, 255, .6);
        margin-left: 20px;
      }
      img {
        width: 78px;
        height: 78px;
      }
      .title {
        font-weight: 400;
      }
    </style>
    <img src='//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
    <div class="loading"> Loading
      <span class="title">GraphQL Playground</span>
    </div>
  </div>
  <script>window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        endpoint: '/graphql',
        subscriptionEndpoint: '/graphql',
      })
    })</script>
</body>

</html>
  `))
	t.Execute(w, httpPort)
}
