# Plugin SDK Developer Guide

Project  : Limoxel  
Category : Plugin Ecosystem  
Document : Plugin SDK Developer Guide  
Version  : 1.0  
Author   : Raj Joshi

---

## Purpose

This document explains how developers can build plugins for the Limoxel ecosystem using the Limoxel Plugin SDK.

The Plugin SDK provides the supported developer-facing contracts, APIs, templates, and development tools required to create plugins that interact with Limoxel through defined interfaces.

The SDK is designed to allow plugin developers to build specialized functionality while using capabilities already provided by Limoxel.

Plugins should interact with Limoxel through the supported SDK rather than depending on private implementation details.

---

## Overview

A Limoxel plugin is an independently developed extension that provides additional functionality within the Limoxel ecosystem.

The Plugin SDK provides the developer-facing interface through which a plugin can:

- Identify itself
- Describe its capabilities
- Participate in the plugin lifecycle
- Access supported repository information
- Work with source-code symbols
- Perform supported searches
- Query supported graph information
- Request supported analysis
- Subscribe to supported events
- Report errors
- Use supported configuration and host context
- Be developed and tested using the provided tooling

The SDK also provides project templates and development tools intended to make plugin development consistent and predictable.

---

## SDK Role

The Plugin SDK forms the supported boundary between a plugin and Limoxel.

```text
Plugin
   |
   +-- Plugin SDK
          |
          +-- Repository API
          +-- Symbol API
          +-- Search API
          +-- Graph API
          +-- Analysis API
          +-- Event API
          +-- Plugin Lifecycle
          +-- Host Context
```

The SDK exposes supported Limoxel functionality through defined contracts.

Plugin developers should use these contracts instead of relying on private Limoxel implementation details.

This separation allows plugins to remain focused on their own functionality while Limoxel remains responsible for the underlying engineering capabilities.

---

## SDK Principles

The Plugin SDK follows several principles.

### Supported Contracts

Plugin developers should build against documented SDK interfaces rather than private platform implementation details.

Supported contracts provide a predictable foundation for plugin development.

---

### Capability Reuse

Plugins consume capabilities already provided by Limoxel.

The SDK does not require plugin developers to reproduce repository analysis, symbol processing, search, graph processing, or other platform capabilities.

---

### Clear Boundaries

The SDK separates plugin functionality from the internal implementation of Limoxel.

A plugin should interact with the platform through the interfaces provided to it.

---

### Explicit Compatibility

Plugins should declare the versions and capabilities they require.

Compatibility information allows the host environment to determine whether a plugin can operate with the available platform and SDK.

---

### Predictable Lifecycle

Plugins participate in a defined lifecycle.

Plugin implementations should initialize their resources, begin active work, and release resources according to the lifecycle contract.

---

### Context and Cancellation

SDK operations use Go contexts where applicable.

Plugin implementations should respect cancellation and avoid continuing work after the relevant operation or lifecycle context has been cancelled.

---

### Resource Responsibility

Plugins are responsible for resources they create or retain.

Event subscriptions, background workers, open resources, and other plugin-owned resources should be released when the plugin stops.

---

## Plugin Development

A plugin normally consists of three conceptual parts:

1. Plugin metadata
2. Plugin implementation
3. Plugin-specific functionality

The plugin metadata identifies the plugin and describes its requirements.

The plugin implementation provides the lifecycle behavior expected by Limoxel.

The plugin-specific functionality implements the actual purpose of the plugin.

A minimal project can be created using the Basic Plugin template.

---

## Plugin Identity and Metadata

Every plugin should have a stable identity.

Plugin metadata communicates information such as:

- Plugin identifier
- Plugin name
- Plugin version
- Publisher
- Description
- Required capabilities
- Compatibility information
- Runtime information where applicable

A plugin identifier should remain stable throughout the lifetime of the plugin.

Changing a plugin identifier represents a different plugin identity rather than a normal version update.

---

## Plugin Manifest

Plugins use a manifest to describe their identity and requirements.

A manifest provides declarative information that can be examined before the plugin is activated.

A representative manifest has the following form:

```json
{
  "schema_version": "1.0.0",
  "id": "com.example.myplugin",
  "name": "My Plugin",
  "version": "1.0.0",
  "publisher": "Example",
  "description": "Example Limoxel plugin.",
  "capabilities": [
    {
      "name": "repository.metadata",
      "version": "1.0.0"
    }
  ]
}
```

The exact fields and accepted values are determined by the supported Plugin Framework and SDK contracts.

Developers should use the SDK and supported tooling rather than relying on undocumented manifest behavior.

---

## Plugin Lifecycle

Plugins participate in the Limoxel plugin lifecycle.

A plugin implementation should distinguish initialization from active execution.

A typical lifecycle consists of:

```text
Create
  |
  v
Initialize
  |
  v
Start
  |
  v
Running
  |
  v
Stop
  |
  v
Released
```

The exact lifecycle transitions are managed by the Limoxel plugin environment.

Plugin implementations should not assume that they control the entire lifecycle themselves.

---

### Initialization

Initialization is used to prepare the plugin for operation.

Initialization should generally be used for tasks such as:

- Retaining supported host context
- Validating plugin-specific configuration
- Preparing internal state
- Preparing resources required for startup

Initialization should not unnecessarily begin long-running work.

---

### Start

Start begins active plugin operation.

Long-running workers, event subscriptions, and other active operations should normally begin during the appropriate active lifecycle phase.

Plugins should return an error when startup cannot be completed successfully.

---

### Stop

Stop terminates active plugin behavior and releases plugin-owned resources.

A plugin should:

- Stop background work
- Unsubscribe from events
- Close resources
- Release temporary state
- Return promptly where possible

Plugin shutdown should be safe and predictable.

---

## Plugin Context

The Plugin SDK provides a controlled context through which plugins interact with Limoxel.

The context may provide access to supported services such as:

- Host information
- Repository capabilities
- Symbol capabilities
- Search capabilities
- Graph capabilities
- Analysis capabilities
- Event capabilities
- Configuration
- Logging and diagnostics where provided

The context is intended to keep plugin interactions within supported SDK boundaries.

Plugins should not attempt to obtain private Limoxel services through implementation-specific mechanisms.

---

## Repository API

The Repository API provides plugins with supported access to repository information.

It is intended for plugins that need to understand the repository being operated on without implementing their own repository processing system.

Depending on the available repository capabilities, plugins can work with information such as:

- Repository identity
- Repository metadata
- Repository statistics
- Files
- Packages
- Repository state

The Repository API should be used when a plugin needs information already maintained by Limoxel.

---

### Repository Information

Repository information provides a plugin with a structured view of the repository.

A plugin may use repository information to determine characteristics such as:

- Repository identity
- Repository state
- Supported languages
- Repository statistics
- Other supported repository metadata

Plugins should not assume that every repository provides every optional piece of information.

---

### Repository Metadata

Repository metadata represents descriptive information associated with a repository.

Plugins can use supported metadata to provide repository-specific functionality without independently discovering the same information.

---

### Repository Statistics

Repository statistics provide structured measurements available from the repository capabilities.

Examples can include counts or summaries related to:

- Files
- Directories
- Packages
- Symbols

The exact statistics available depend on the supported repository capabilities.

---

### File Access

Where the Repository API provides file access, plugins should use the supported file interfaces rather than accessing Limoxel's internal repository structures.

File paths supplied to an API should follow the path rules defined by the SDK.

Plugins should validate and handle errors returned by file operations.

---

## Symbol API

The Symbol API provides access to supported source-code symbol information.

It allows plugins to work with symbols without implementing their own source-code parsing and symbol indexing systems.

Depending on the supported capabilities, symbol operations can include:

- Symbol lookup
- Symbol search
- Symbol references
- Symbol relationships
- Symbol hierarchy
- Symbol documentation

---

### Symbol Lookup

Symbol lookup resolves a symbol using supported symbol identifiers or names.

A successful lookup can provide structured information about the symbol.

Plugins should handle unresolved symbols explicitly rather than assuming that every requested symbol exists.

---

### Finding Symbols

The Symbol API can be used to find symbols according to supported patterns and scopes.

Plugin developers should use the available filtering and pagination mechanisms where provided.

---

### Symbol References

Where supported, plugins can request references to a symbol.

This can be useful for plugins that need to understand how a symbol is used across a codebase.

---

### Symbol Relationships

Symbol relationships provide structured information about connections between symbols.

Plugins should treat relationship information according to the guarantees of the API rather than assuming that every possible relationship is available.

---

### Symbol Documentation

Where documentation is available through the symbol system, plugins can retrieve supported documentation associated with a symbol.

---

## Search API

The Search API provides plugins with access to Limoxel's supported search capabilities.

Search allows plugins to find information without implementing a separate search engine.

Supported search domains can include:

- Symbols
- Files
- Packages
- Documentation
- Other supported repository information

---

### Search Requests

A search request can contain information such as:

- Query
- Search domain
- Scope
- Filters
- Result limits
- Other supported search options

The plugin should use the request structure defined by the SDK.

---

### Search Results

Search results provide structured information corresponding to the requested search.

Plugins should not assume that a search always returns results.

An empty result set is different from a failed search operation.

---

### Search Limits

Plugins should use supported limits when requesting potentially large result sets.

Large unbounded queries should be avoided when a narrower query can provide the required information.

---

## Graph API

The Graph API provides access to supported relationships represented within Limoxel's engineering knowledge graph.

It allows plugins to reason about relationships without maintaining a separate graph representation.

Supported operations can include:

- Node lookup
- Relationship lookup
- Neighbor queries
- Traversal
- Filtering
- Supported graph export

---

### Graph Nodes

A graph node represents an entity known to the graph system.

Depending on the graph model, nodes can represent engineering entities such as:

- Symbols
- Files
- Packages
- Repositories
- Other supported entities

Plugins should use the node types defined by the SDK.

---

### Graph Relationships

Graph relationships describe supported connections between entities.

Plugins can use relationship queries to understand connections within the engineering system.

---

### Graph Traversal

Traversal operations allow plugins to navigate supported graph relationships.

Plugins should specify appropriate traversal constraints when available to avoid unnecessarily large traversals.

---

## Analysis API

The Analysis API exposes supported Limoxel analysis capabilities to plugins.

It allows plugins to consume analysis results without implementing a second analysis engine.

Supported analysis areas can include:

- Architecture information
- Dependency information
- Repository analysis
- Engineering quality information
- Other supported analysis capabilities

The available analysis operations depend on the capabilities exposed by the Limoxel environment.

---

### Analysis Requests

Analysis operations should be invoked with the appropriate context and request parameters.

Plugins should handle:

- Successful analysis
- Invalid requests
- Unsupported operations
- Cancellation
- Runtime failures

appropriately.

---

### Analysis Results

Analysis results should be treated as structured data.

Plugins should not assume that a result is available for every repository or every requested analysis.

---

## Event API

The Event API allows plugins to interact with supported Limoxel events.

Events provide a mechanism for plugins to respond to changes or activities exposed by the platform.

Depending on the supported event model, plugins may work with events associated with:

- Repository activity
- Analysis activity
- Plugin lifecycle
- Other supported platform events

---

### Event Subscription

A plugin can subscribe to supported events through the Event API.

A subscription should be retained by the plugin so that it can be released during shutdown.

A typical subscription pattern is:

```go
subscription, err := events.Subscribe(ctx, filter, handler)
if err != nil {
    return err
}

pluginSubscription = subscription
```

The exact API depends on the SDK version in use.

---

### Event Handling

Event handlers should:

- Process events efficiently
- Respect cancellation
- Handle errors
- Avoid unnecessary blocking
- Protect shared plugin state
- Release resources when the plugin stops

Plugins should not assume that event handlers are serialized unless the SDK explicitly guarantees that behavior.

---

### Unsubscribing

Plugins should unsubscribe from events during shutdown.

For example:

```go
if subscription != nil {
    _ = subscription.Unsubscribe()
}
```

The exact cleanup behavior is determined by the SDK contract.

Event subscriptions should never be treated as permanently owned resources.

---

## Errors

SDK operations return errors when an operation cannot be completed successfully.

Plugins should inspect and handle errors rather than ignoring them.

Common categories of failure can include:

- Invalid input
- Unsupported capability
- Missing resource
- Incompatible version
- Runtime failure
- Transport failure
- Timeout
- Cancellation
- Repository failure
- Analysis failure
- Event failure

The SDK may provide structured error information that allows callers to distinguish these conditions.

---

### Error Wrapping

Plugin code should preserve useful error context.

For example:

```go
result, err := repository.Info(ctx)
if err != nil {
    return fmt.Errorf("retrieve repository information: %w", err)
}
```

Error messages should describe the operation that failed without exposing unnecessary internal implementation details.

---

### Context and Cancellations

SDK operations that accept `context.Context` should receive the relevant context from the caller.

For example:

```go
result, err := search.Search(ctx, request)
```

Plugins should propagate cancellation rather than creating unrelated long-lived contexts.

Long-running operations should stop when their context is cancelled.

Background workers should also have a controlled cancellation mechanism.

---

## Timeouts

Plugins performing operations that can block should use appropriate timeout behavior.

For example:

```go
ctx, cancel := context.WithTimeout(parent, timeout)
defer cancel()

result, err := repository.Info(ctx)
```

Timeout values should be selected according to the actual operation.

Plugins should not use arbitrary very short timeouts that make normal operations unreliable.

Likewise, plugins should avoid indefinite blocking when an operation has a reasonable completion boundary.

---

## Concurrency

Plugins may perform multiple operations concurrently when supported by the SDK.

Plugin developers should ensure that their own shared state is synchronized appropriately.

For example:

```go
type PluginState struct {
    mu      sync.RWMutex
    running bool
}
```

Synchronization should protect plugin-owned state without unnecessarily serializing independent operations.

---

### Concurrent API Calls

Independent SDK operations can be performed concurrently where the relevant API contract permits it.

Plugins should not assume that an API is safe for concurrent use unless that behavior is documented or otherwise guaranteed by the SDK.

---

### Concurrent Event Handling

Event handlers should be written with the actual event delivery behavior in mind.

If multiple handlers or events can execute concurrently, shared state must be protected appropriately.

---

## Resource Management

Plugin resources should have clear ownership.

Resources can include:

- Event subscriptions
- Goroutines
- Timers
- Files
- Temporary resources
- Network connections
- Plugin-owned state
- Other externally allocated resources

A plugin should release resources when they are no longer needed.

---

### Background Workers

Background workers should have a controlled lifecycle.

A common pattern is:

```go
workerCtx, cancel := context.WithCancel(ctx)
defer cancel()

go runWorker(workerCtx)
```

The worker should observe cancellation and terminate cleanly.

Plugins should not leave background goroutines running after the plugin has stopped.

---

### Event Subscriptions

Event subscriptions should be stored so that they can be released during shutdown.

For example:

```go
type Plugin struct {
    subscription Subscription
}
```

The plugin should release the subscription as part of its shutdown behavior.

---

## Configuration

Plugins may require configuration specific to their functionality.

Configuration should be:

- Explicit
- Validated
- Documented
- Limited to what the plugin requires

Plugins should provide useful errors when required configuration is invalid or missing.

Sensitive configuration values should not be written to logs unnecessarily.

---

## Logging and Diagnostics

Plugins should provide useful diagnostics for their own operations.

Good diagnostic information can include:

- Operation being performed
- Plugin-specific identifiers
- Relevant non-sensitive parameters
- Failure context
- Lifecycle events
- Recovery information

Sensitive information should not be logged.

Plugins should prefer structured logging where the SDK or host environment provides it.

---

## Compatibility

Plugin compatibility should be treated as an explicit engineering concern.

Plugins should declare the platform and capability requirements they depend upon.

Compatibility information may include:

- Plugin version
- SDK version
- Limoxel compatibility
- Capability versions
- Dependency requirements

Plugins should avoid depending on undocumented behavior.

---

### Versioning

Plugin versions should follow the versioning rules supported by the Limoxel ecosystem.

Semantic versioning uses the familiar:

```text
MAJOR.MINOR.PATCH
```

format where applicable.

Developers should increment versions according to the compatibility impact of their changes.

---

### Capability Versions

Capabilities can have their own version information.

A plugin should request the capability version it actually requires.

Plugins should avoid declaring unnecessarily broad compatibility requirements.

---

### Compatibility Failures

When a required capability or version is unavailable, the plugin should report the incompatibility clearly.

A plugin should not silently substitute an incompatible implementation.

---

## Plugin Templates

The Plugin SDK provides canonical templates for common plugin development patterns.

The available templates are:

1. Basic
2. CLI
3. Repository
4. Intelligence
5. Enterprise

The templates provide starting points rather than restricting what a plugin can ultimately implement.

---

### Basic Plugin Template

The Basic template provides the smallest practical starting point for plugin development.

It is appropriate when a developer wants to begin with the fundamental plugin lifecycle and add functionality incrementally.

The template demonstrates the basic plugin structure and lifecycle.

---

### CLI Plugin Template

The CLI template provides a starting point for plugins that expose command-oriented functionality.

It is intended for plugin functionality that naturally fits a command-line workflow.

The template does not replace Limoxel's command-line infrastructure.

---

### Repository Plugin Template

The Repository template provides a starting point for plugins that consume repository information.

It can demonstrate interactions with supported repository APIs and related events.

It is suitable for plugins whose primary purpose involves understanding or processing repository information.

---

### Intelligence Plugin Template

The Intelligence template provides a starting point for plugins that use engineering intelligence capabilities.

It can demonstrate supported APIs such as:

- Symbols
- Search
- Graph
- Analysis

The template does not require the plugin to implement its own intelligence engine.

---

### Enterprise Plugin Template

The Enterprise template provides a structured developer scaffold for plugins intended for organizational environments.

It can demonstrate conventions such as:

- Configuration
- Structured diagnostics
- Lifecycle handling
- Compatibility metadata
- Operational organization

The template is a development starting point. Organizational deployment and management behavior depends on the environment in which the plugin is used.

---

## Plugin Generator

The Plugin SDK provides generator tooling for creating plugin projects from the supported templates.

The generator can create a project structure based on information such as:

- Template
- Plugin identifier
- Plugin name
- Version
- Publisher
- Description
- Output directory

A typical generator workflow is:

```text
Select template
      |
      v
Provide plugin metadata
      |
      v
Validate project information
      |
      v
Generate project
      |
      v
Build and test plugin
```

Generated projects should provide a usable starting point rather than an empty directory.

---

### Generator Input Validation

Generator input should be validated before project creation.

Validation should protect against:

- Invalid plugin identifiers
- Invalid project names
- Unsafe output paths
- Invalid versions
- Invalid metadata

Developers should correct invalid input rather than bypassing generator validation.

---

### Generated Projects

Generated projects should follow the structure expected by the selected template.

After generation, developers should inspect the generated project and customize it for their plugin's actual purpose.

Generated code is a starting point and remains the responsibility of the plugin developer.

---

## Build Tooling

Plugin build tooling assists developers in validating and compiling plugin projects.

The build process should:

1. Validate the plugin project
2. Validate required plugin metadata
3. Invoke the appropriate build process
4. Report build failures
5. Produce the plugin executable or build artifact

The build tooling uses the supported development toolchain rather than implementing a separate compiler.

A typical workflow is:

```text
Plugin Project
      |
      v
Project Validation
      |
      v
Manifest Validation
      |
      v
Compilation
      |
      v
Plugin Artifact
```

---

## Packaging Tooling

Packaging tooling creates a distributable plugin artifact from a valid plugin project.

A package can contain the information required to identify and execute the plugin, including:

- Plugin manifest
- Plugin executable
- Required plugin documentation
- Other supported package contents

Packaging should preserve the plugin's identity and version information.

Where deterministic packaging is supported, equivalent inputs should produce consistent package contents.

---

## Debugging Tooling

The SDK provides developer-oriented tooling to assist with local plugin development and diagnosis.

Debugging support can help developers inspect:

- Plugin startup
- Plugin lifecycle
- Configuration
- Runtime communication
- Diagnostic information
- Plugin failures

Debugging tools are intended for plugin development and troubleshooting.

They do not replace a general-purpose debugger or development environment.

---

## Testing Tooling

The Plugin SDK provides testing support for plugin developers.

Testing facilities can help developers test plugin behavior against supported SDK contracts.

A plugin test can verify:

- Initialization
- Startup
- Shutdown
- Repository API usage
- Symbol API usage
- Search API usage
- Graph API usage
- Analysis API usage
- Event subscriptions
- Error handling
- Cancellation
- Resource cleanup

---

### Test Harness

A test harness can provide controlled host behavior for plugin tests.

The purpose of a test harness is to allow plugin behavior to be exercised without requiring every test to depend on a complete external environment.

Test helpers should remain consistent with the supported SDK contracts.

---

### Testing Real Behavior

Plugin tests should verify behavior rather than merely checking that interfaces exist.

For example, a repository plugin test should exercise a repository operation and verify its result.

A search plugin test should submit a search request and verify the returned results.

An event test should verify that an event is delivered and that the subscription can be released.

---

## Examples

A simple plugin can follow a structure similar to:

```go
package main

import (
    "context"

    "github.com/unhield/limoxel/plugin"
)

type ExamplePlugin struct {
    ctx plugin.Context
}

func (p *ExamplePlugin) ID() string {
    return "com.example.plugin"
}

func (p *ExamplePlugin) Init(ctx context.Context, host plugin.Host) error {
    // Initialize plugin state using supported SDK contracts.
    return nil
}

func (p *ExamplePlugin) Start(ctx context.Context) error {
    // Begin active plugin work.
    return nil
}

func (p *ExamplePlugin) Stop(ctx context.Context) error {
    // Release plugin-owned resources.
    return nil
}
```

The exact interfaces and supporting types should be taken from the SDK version used by the plugin project.

---

## Repository Example

A repository-oriented plugin can use the repository API to request information already available from Limoxel.

A conceptual example is:

```go
info, err := repository.Info(ctx)
if err != nil {
    return fmt.Errorf("retrieve repository information: %w", err)
}

processRepository(info)
```

The plugin should consume the returned information rather than implementing an independent repository scanner.

---

## Search Example

A plugin can use the Search API to query supported engineering information.

A conceptual pattern is:

```go
results, err := search.Search(ctx, request)
if err != nil {
    return fmt.Errorf("search repository: %w", err)
}

for _, result := range results {
    processResult(result)
}
```

The plugin should use the search capabilities exposed by the SDK instead of maintaining another search implementation.

---

## Event Example

A plugin that responds to repository events can subscribe during startup and release its subscription during shutdown.

A conceptual pattern is:

```go
subscription, err := events.Subscribe(ctx, filter, func(
    eventCtx context.Context,
    event Event,
) error {
    return handleEvent(eventCtx, event)
})
if err != nil {
    return err
}

pluginSubscription = subscription
```

During shutdown:

```go
if pluginSubscription != nil {
    return pluginSubscription.Unsubscribe()
}
```

The exact event types and subscription signatures depend on the SDK version.

---

## Plugin Tooling Workflow

A typical plugin development workflow is:

```text
Create Plugin
      |
      v
Select Template
      |
      v
Generate Project
      |
      v
Implement Plugin
      |
      v
Use SDK APIs
      |
      v
Write Tests
      |
      v
Build Plugin
      |
      v
Package Plugin
      |
      v
Run and Diagnose
      |
      v
Install / Activate
```

Developers should validate the plugin throughout development rather than waiting until packaging.

---

## Recommended Development Workflow

### 1. Define the Plugin Purpose

Start by defining the specific engineering problem the plugin solves.

A focused plugin is generally easier to understand, test, maintain, and evolve.

---

### 2. Choose an Appropriate Template

Select the template that most closely matches the plugin's primary purpose.

The template should provide a starting structure, not dictate the final implementation.

---

### 3. Generate the Project

Use the plugin generator to create the initial project.

Verify the generated metadata before beginning implementation.

---

### 4. Implement the Lifecycle

Implement initialization, startup, and shutdown behavior.

Ensure every resource created by the plugin has a corresponding cleanup path.

---

### 5. Use Supported APIs

Use the Repository, Symbol, Search, Graph, Analysis, and Event APIs where appropriate.

Avoid implementing functionality that Limoxel already provides.

---

### 6. Handle Errors

Treat errors explicitly.

Provide enough context to make failures understandable without exposing sensitive information.

---

### 7. Handle Cancellation

Propagate contexts to SDK operations and stop background work when cancellation occurs.

---

### 8. Test the Plugin

Test normal operation as well as:

- Invalid input
- Missing data
- API failures
- Cancellation
- Shutdown
- Event cleanup
- Concurrent operations where applicable

---

### 9. Build the Plugin

Use the supported build tooling to validate the project and produce its executable artifact.

---

### 10. Package the Plugin

Create the plugin package using the supported packaging tooling.

Verify the package metadata before distributing it.

---

## Best Practices

### Keep Plugins Focused

A plugin should have a clear purpose.

Avoid combining unrelated functionality into a single plugin merely because the functionality can technically coexist.

---

### Prefer Existing Capabilities

Before implementing a new capability, determine whether Limoxel already exposes the information through the Plugin SDK.

Using existing capabilities reduces duplication and keeps plugin behavior consistent with the platform.

---

### Keep Lifecycle Operations Explicit

Plugins should make their resource ownership clear.

Initialization, startup, and shutdown should each have a well-defined responsibility.

---

### Clean Up Resources

Every plugin-owned resource should have a corresponding cleanup path.

This is particularly important for:

- Goroutines
- Event subscriptions
- Files
- Timers
- Connections
- Temporary resources

---

### Propagate Contexts

Use the context supplied by the caller whenever an SDK operation accepts one.

Do not replace an operation's context with an unrelated background context unless there is a deliberate reason to do so.

---

### Respect Cancellation

Long-running operations should stop when their context is cancelled.

Background workers should also have a controlled cancellation mechanism.

---

### Avoid Blocking Event Handlers

Event handlers should complete efficiently where possible.

If substantial work is required, the plugin should use an appropriate asynchronous design while retaining clear ownership and cancellation behavior.

---

### Protect Shared State

When multiple operations can access plugin-owned mutable state concurrently, synchronize that state appropriately.

Do not assume serialized execution unless the relevant contract explicitly provides it.

---

### Validate Inputs

Plugin input should be validated before processing.

Validation is particularly important for:

- Paths
- Identifiers
- Configuration
- Search parameters
- External input
- Plugin-specific commands

---

### Keep Compatibility Explicit

Declare the capabilities and versions required by the plugin.

Avoid depending on undocumented behavior that may change independently of supported SDK contracts.

---

### Keep Diagnostics Useful

Diagnostics should help developers understand what the plugin was doing when an error occurred.

Avoid logging secrets, credentials, tokens, or other sensitive information.

---

### Prefer Deterministic Behavior

When equivalent inputs and environments produce equivalent results, plugin behavior should remain consistent.

Avoid unnecessary dependence on:

- Unordered iteration
- Hidden global state
- Uncontrolled external state
- Timing assumptions

---

### Keep APIs Minimal

Plugin APIs should expose only the functionality needed by plugin developers.

A smaller, clearer API is easier to understand and maintain.

---

## Plugin Security Practices

The Plugin SDK does not remove the responsibility of plugin developers to write safe software.

Plugins should:

- Validate inputs
- Avoid unnecessary privileges
- Protect sensitive configuration
- Avoid unsafe command construction
- Handle paths carefully
- Validate external data
- Handle failures safely
- Release resources
- Avoid exposing sensitive diagnostic information

Plugins should not assume that their code is automatically safe simply because it uses the SDK.

---

## Working with Files and Paths

Plugins that work with repository files should use supported repository and file APIs where available.

When a plugin accepts a user-provided or externally supplied path, it should validate the path before using it.

Plugins should avoid constructing operating-system commands directly from untrusted path or argument values.

---

## Working with External Processes

A plugin that needs to execute an external process should validate the executable and arguments carefully.

Arguments should be passed as structured process arguments rather than concatenated into shell commands.

For example:

```go
cmd := exec.CommandContext(ctx, executable, arg1, arg2)
```

rather than constructing a shell command from an untrusted string.

Plugins should also handle process failures and cancellation correctly.

---

## Plugin Dependencies

Plugins should declare dependencies on supported capabilities where required.

Dependencies should be explicit rather than hidden in plugin implementation behavior.

A plugin should fail clearly when a required capability is unavailable.

Optional functionality should be designed so that its absence can be handled safely when appropriate.

---

## Plugin Documentation

Every published plugin should provide documentation appropriate to its users.

At minimum, plugin documentation should explain:

- Plugin purpose
- Supported functionality
- Requirements
- Configuration
- Usage
- Compatibility
- Known limitations
- Troubleshooting information

Documentation should match the actual behavior of the plugin.

---

## Plugin README

A plugin README should normally contain:

```text
Plugin name
Purpose
Capabilities
Requirements
Installation
Configuration
Usage
Compatibility
Examples
Limitations
Troubleshooting
License
```

The exact structure can vary according to the plugin.

---

## Compatibility and Evolution

Plugins should be designed so that they can evolve without unnecessarily breaking existing users.

When changing a plugin:

1. Determine whether the change is compatible.
2. Update the plugin version appropriately.
3. Update documentation.
4. Update examples.
5. Update tests.
6. Communicate meaningful compatibility changes to users.

Breaking changes should be deliberate and clearly documented.

---

## Plugin Testing Checklist

Before distributing a plugin, verify:

- [ ] Plugin identity is valid.
- [ ] Plugin metadata is complete.
- [ ] Required capabilities are declared.
- [ ] Initialization succeeds under normal conditions.
- [ ] Startup succeeds under normal conditions.
- [ ] Shutdown releases plugin-owned resources.
- [ ] Repository operations handle failures.
- [ ] Symbol operations handle missing symbols.
- [ ] Search operations handle empty results and failures.
- [ ] Graph operations handle invalid queries.
- [ ] Analysis operations handle failures and cancellation.
- [ ] Event subscriptions are released.
- [ ] Context cancellation is respected.
- [ ] Concurrent access to shared state is safe.
- [ ] Configuration is validated.
- [ ] Sensitive information is not unnecessarily logged.
- [ ] The plugin builds successfully.
- [ ] The plugin package contains the required metadata.
- [ ] Documentation matches the implementation.

---

## Plugin Distribution Preparation

Before distributing a plugin, verify:

- Plugin identifier is stable.
- Plugin version is correct.
- Manifest is valid.
- Required capabilities are declared.
- Compatibility information is accurate.
- Package contents are correct.
- Executable artifacts are present.
- Documentation is included where required.
- Tests have passed.
- No unnecessary development artifacts are included.

---

## Troubleshooting

## Plugin Does Not Start

Check:

1. Plugin metadata
2. Plugin identifier
3. Version information
4. Required capabilities
5. Configuration
6. Initialization errors
7. Startup errors
8. Runtime diagnostics

---

### API Operation Fails

Check:

1. Context cancellation
2. Input validation
3. Capability availability
4. Plugin compatibility
5. Repository state
6. Runtime state
7. Returned error information

Do not assume that an API failure indicates a failure of the plugin runtime itself.

---

### Events Are Not Received

Check:

1. Event subscription succeeded.
2. The event filter matches the intended event.
3. The plugin remains active.
4. The subscription has not been cancelled.
5. The event is supported in the current environment.
6. The plugin has not already begun shutdown.

---

### Plugin Does Not Shut Down Cleanly

Check:

1. Background goroutines
2. Event subscriptions
3. Open files
4. Timers
5. External connections
6. Child processes
7. Plugin-owned resources

Every long-lived resource should have an explicit shutdown path.

---

## API Design Guidance

Plugin developers should prefer APIs that are:

- Small
- Explicit
- Typed
- Context-aware
- Testable
- Versionable
- Predictable

Avoid exposing unnecessary implementation details through plugin-specific APIs.

If a plugin provides its own API to other plugins, it should document the contract with the same care as a public SDK interface.

---

## Plugin Quality

A high-quality plugin should be:

- Focused
- Predictable
- Well documented
- Tested
- Compatible with supported SDK versions
- Conscious of resource ownership
- Safe in concurrent operation
- Explicit about errors
- Respectful of cancellation
- Independent of private Limoxel implementation details

The Plugin SDK provides the foundation for these practices, but the quality of an individual plugin remains the responsibility of its developer.

---

## Summary

The Limoxel Plugin SDK provides the supported developer-facing foundation for building plugins.

It provides:

- Plugin contracts
- Plugin lifecycle integration
- Host context
- Repository APIs
- Symbol APIs
- Search APIs
- Graph APIs
- Analysis APIs
- Event APIs
- Plugin templates
- Project generation
- Build tooling
- Packaging tooling
- Debugging support
- Testing support
- Compatibility guidance
- Development best practices

The SDK allows developers to extend Limoxel while using the engineering capabilities already available through the platform.

Plugins should remain focused on the functionality they provide and should use supported SDK contracts rather than depending on private implementation details.

---

## Authority

This document is the authoritative public developer guide for the Limoxel Plugin SDK.

It describes the supported concepts, development interfaces, APIs, templates, tooling, and development practices associated with building Limoxel plugins.

---

## Applicability

This document applies to developers building plugins for the Limoxel ecosystem and to plugins using the supported Limoxel Plugin SDK.

It describes the public developer experience and supported plugin-development concepts without defining private implementation details of Limoxel.

---

## Change Policy

This document is maintained as part of the Limoxel public documentation.

Changes should preserve clarity, consistency, and compatibility with the supported Plugin SDK.

Existing documented SDK behavior should not be changed without appropriate compatibility consideration and corresponding documentation updates.

New SDK capabilities and developer tooling should be documented additively whenever reasonably possible.
