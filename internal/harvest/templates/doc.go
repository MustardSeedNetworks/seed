// Package templates provides report template management and rendering capabilities
// for the harvest module. It includes:
//
//   - Template: A report template definition with sections, formats, and metadata
//   - Renderer: Template rendering engine with built-in helper functions
//   - Registry: Thread-safe template storage and retrieval
//   - IDValidator: Template ID validation and sanitization
//
// This package is designed to be used by the harvest module for generating
// network diagnostic reports in various formats (PDF, HTML, CSV, JSON, etc.).
package templates
