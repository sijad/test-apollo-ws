import { ApolloClient, InMemoryCache } from "apollo-boost";
import { split } from "apollo-link";
import { HttpLink } from "apollo-link-http";
import { WebSocketLink } from "apollo-link-ws";
import { getMainDefinition } from "apollo-utilities";

const httpLink = new HttpLink({
  uri: "http://localhost:3000/graphql"
});

const wsLink = new WebSocketLink({
  uri: `ws://localhost:3000/graphql`,
  options: { reconnect: true }
});

const link = split(
  ({ query }) => {
    const definition = getMainDefinition(query);
    return (
      definition.kind === "OperationDefinition" &&
      definition.operation === "subscription"
    );
  },
  wsLink,
  httpLink
);

export const client = new ApolloClient({
  link,
  cache: new InMemoryCache().restore({})
});
