# Interactive Sandbox

The Interactive Sandbox lets you explore Consize's optimization workflow without setting up a production Kubernetes environment.

It is the fastest way to understand what Consize does before installing it in your own cluster.

## What you can explore

The sandbox demonstrates the core Consize workflow:

```text id="4r8m2c"
Observe
   ↓
Analyze
   ↓
Recommend
   ↓
Review
   ↓
Apply
   ↓
Verify
```

You can see how Consize identifies an optimization opportunity and evaluates the proposed change before it is applied.

## Why use the sandbox?

Use the sandbox if you want to:

* Understand how Consize works
* See a rightsizing recommendation
* Explore the safety checks
* Understand the approval and application flow
* Try Consize before installing it

## No production access required

The sandbox is designed for exploration.

You do not need to connect it to your production Kubernetes cluster or provide production credentials.

This makes it useful for evaluating the workflow before introducing Consize into a real environment.

## From the sandbox to production

Once you understand the workflow, you can install Consize in your own Kubernetes environment.

The recommended path is:

```text id="7f3m1a"
Interactive Sandbox
       ↓
Quickstart
       ↓
Production Installation
       ↓
Recommendations
       ↓
Controlled automation
```

## Next steps

* [Quickstart](quickstart.md)
* [Production Installation](installation.md)
* [Kubernetes Rightsizing](../guides/rightsizing.md)
* [The Safety Net](../concepts/safety-net.md)
