import React from "react";
import { gql } from "apollo-boost";
import { useSubscription } from "@apollo/react-hooks";
import logo from "./logo.svg";
import "./App.css";

const sub = gql`
  subscription {
    helloSaid {
      id
      msg
    }
  }
`;

function App() {
  const a = useSubscription(sub, {
    onSubscriptionComplete: () => alert("completed")
  });

  return (
    <div className="App">
      <header className="App-header">
        <img src={logo} className="App-logo" alt="logo" />
        <p>
          Edit <code>src/App.js</code> and save to reload.
        </p>
        <a
          className="App-link"
          href="https://reactjs.org"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn React
        </a>
      </header>
    </div>
  );
}

export default App;
