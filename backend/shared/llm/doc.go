// Package llm provides shared abstractions over LLM providers
// (currently Doubao via Volcengine Ark) used by product/auction services.
//
// Lives in backend/shared/llm/ as an independent Go module so that
// services can import it via `replace shared/llm => ../shared/llm`,
// keeping cross-service code reuse at build time without introducing
// a separate microservice.
package llm
