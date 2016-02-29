$(function (global) {
  /**
   * This GraphiQL example illustrates how to use some of GraphiQL's props
   * in order to enable reading and updating the URL parameters, making
   * link sharing of queries a little bit easier.
   *
   * This is only one example of this kind of feature, GraphiQL exposes
   * various React params to enable interesting integrations.
   */

  // Parse the search string to get url parameters.
  var search = window.location.search;
  var parameters = {};
  search.substr(1).split('&').forEach(function (entry) {
    var eq = entry.indexOf('=');
    if (eq >= 0) {
      parameters[decodeURIComponent(entry.slice(0, eq))] =
          decodeURIComponent(entry.slice(eq + 1));
    }
  });

  // if variables was provided, try to format it.
  if (parameters.variables) {
    try {
      parameters.variables =
          JSON.stringify(JSON.parse(query.variables), null, 2);
    } catch (e) {
      // Do nothing
    }
  }

  // When the query and variables string is edited, update the URL bar so
  // that it can be easily shared
  function onEditQuery(newQuery) {
    parameters.query = newQuery;
    updateURL();
  }

  function onEditVariables(newVariables) {
    parameters.variables = newVariables;
    updateURL();
  }

  function updateURL() {
    var newSearch = '?' + Object.keys(parameters).map(function (key) {
          return encodeURIComponent(key) + '=' +
              encodeURIComponent(parameters[key]);
        }).join('&');
    history.replaceState(null, null, newSearch);
  }

  // Defines a GraphQL fetcher using the fetch API.
  function graphQLFetcher(graphQLParams) {
    return fetch(window.location.origin + '/admin/graphql', {
      method: 'post',
      headers: {
        'Content-Type': 'application/json',
        "Authorization": "Basic eyJpZCI6MjUsImN1c3RvbWVyX2lkIjoiYTFkZTUzZDgtODk3NC0xMWU1LTllN2ItZjM0OWZiNmZhMDQwIiwiZW1haWwiOiJjb21wdXRlckBtYXJrbWFydC5pbiIsIm5hbWUiOiJNYXJrIE1hcnRpbiIsInZlcmlmaWVkIjp0cnVlLCJhZG1pbiI6dHJ1ZSwiYWN0aXZlIjp0cnVlLCJjcmVhdGVkX2F0IjoxNDQ3MzU2OTcyOTU2LCJ1cGRhdGVkX2F0IjoxNDQ3MzY3NjkxMzE2fQ=="
      },
      body: JSON.stringify(graphQLParams)
    }).then(function (response) {
      return response.json()
    });
  }

  var custInput = React.createElement("input", {
    type: "text",
    placeholder: "Customer ID"
  });

  global.renderGraphiql = function (elem) {

    // Render <GraphiQL /> into the body.
    var toolbar = React.createElement(GraphiQL.Toolbar, {}, [
      "Gimme a customer ID ",
      custInput
    ]);

    React.render(
        React.createElement(GraphiQL, {
          fetcher: graphQLFetcher,
          query: parameters.query,
          variables: parameters.variables,
          onEditQuery: onEditQuery,
          onEditVariables: onEditVariables,
          defaultQuery:
            "# Welcome to GraphiQL\n" +
            "#\n" +
            "# GraphiQL is an in-browser IDE for writing, validating, and\n" +
            "# testing GraphQL queries.\n" +
            "#\n" +
            "# Type queries into this side of the screen, and you will\n" +
            "# see intelligent typeaheads aware of the current GraphQL type schema and\n" +
            "# live syntax and validation errors highlighted within the text.\n" +
            "#\n" +
            "# To bring up the auto-complete at any point, just press Ctrl-Space.\n" +
            "#\n" +
            "# Press the run button above, or Cmd-Enter to execute the query, and the result\n" +
            "# will appear in the pane to the right.\n\n" +
            "query checksQuery {\n  checks {\n    id\n    name\n    results {\n      passing\n" +
            "      target {\n        name\n        type\n        id\n        address\n      }\n    }\n  }\n}"
        }, toolbar),
        elem
    );
  }
}(window))
