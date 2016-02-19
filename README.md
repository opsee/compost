GET /ql?query={check(id:\"badead\"){target results}}
GET /ql?query={group(name:\"potato+elb\"){name instances{name }}}

POST /checks ...


GET /graphql
query groupQuery {
        group(name: "potato+elb") {
                name
                instance_count
                instances {
                        name
                        results {
                                responses {
                                        response
                                }
                        }
                }
        }
}

POST /graphql
mutation checkMutation {
        createCheck(...)
}
