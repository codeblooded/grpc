// The orch package provides a library for orchestrating sessions on kubernetes. Users should
// construct a Controller and queue sessions with its Schedule method. The Controller will configure
// many internal types and structures to communicate with kubernetes, monitor the health of test
// components and limit the number of running sessions.
package orch
