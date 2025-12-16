# AI Tools Strategy for LuminetIQ Development

**Document Version:** 1.0
**Last Updated:** 2025-12-15
**Purpose:** Define which AI tools/models to use for different development tasks

---

## Executive Summary

**TL;DR:**
- **Opus 4.5:** Complex architecture, strategic planning, code review ($$$ but worth it)
- **Sonnet 4.5:** Daily coding, debugging, refactoring (best bang/buck)
- **Gemini 2.0 Flash:** Fast code completion, documentation, testing (free tier!)
- **GitHub Copilot:** Real-time autocomplete in IDE (essential)
- **ChatGPT (o1/o1-pro):** Reasoning-heavy tasks, algorithm design, math

**Monthly Cost Estimate:** $200-400/month (all tools combined)

**ROI:** 10x developer productivity = 40-hour tasks done in 4 hours

---

## 1. Claude AI (Anthropic)

### 1.1 Claude Opus 4.5

**Capabilities:**
- **Intelligence:** Highest reasoning ability (PhD-level)
- **Context:** 200K tokens (entire codebase fits)
- **Strengths:**
  - Complex system architecture design
  - Strategic business planning
  - Code review (catches subtle bugs)
  - Multi-file refactoring
  - Research & analysis (market, competitive)

**Best Use Cases for LuminetIQ:**

| Task | Why Opus | Example |
|------|----------|---------|
| **Architecture Design** | Needs deep reasoning, multi-system thinking | "Design time-series database schema for 10K devices × 30 days of metrics" |
| **Business Strategy** | Nuanced analysis, considers trade-offs | "Should we target healthcare or SMB first?" |
| **Code Review** | Catches edge cases, security issues | "Review this packet processing code for memory leaks and race conditions" |
| **Compliance Mapping** | Complex regulatory requirements | "Map our features to HIPAA §164.312(e)(1) with audit evidence" |
| **Competitive Analysis** | Deep industry knowledge, strategic thinking | "Compare our WiFi planning to Ekahau—where do we win/lose?" |

**When NOT to Use Opus:**
- ❌ Simple code completion (overkill, expensive)
- ❌ Boilerplate generation (Sonnet is fine)
- ❌ Quick debugging (Sonnet or Gemini faster/cheaper)
- ❌ Repetitive tasks (use Copilot)

**Cost:**
- **API:** $15/M input tokens, $75/M output tokens
- **Pro Subscription:** $20/month (5x rate limits)
- **Estimated Monthly:** $100-200 for strategic use (10-20 complex tasks)

**Recommendation: Use for 20% of tasks (high-value, complex work)**

---

### 1.2 Claude Sonnet 4.5

**Capabilities:**
- **Intelligence:** Very high (90% of Opus, 1/5th the cost)
- **Speed:** 2-3x faster than Opus
- **Context:** 200K tokens
- **Strengths:**
  - Daily coding tasks (80% of dev work)
  - Debugging
  - Refactoring
  - Testing
  - Documentation

**Best Use Cases for LuminetIQ:**

| Task | Why Sonnet | Example |
|------|------------|---------|
| **Feature Implementation** | Fast, accurate, cost-effective | "Implement DHCP timing analysis with phase-by-phase breakdown" |
| **Debugging** | Quick root cause identification | "Why is this goroutine leaking? Here's the code..." |
| **Refactoring** | Safe transformations, maintains behavior | "Refactor this 500-line function into smaller composable functions" |
| **Unit Tests** | Generates comprehensive test cases | "Write tests for this device classification algorithm" |
| **API Endpoints** | Boilerplate + business logic | "Add REST endpoint for vulnerability scanning" |
| **Documentation** | Clear, concise, accurate | "Document this WiFi scanning package with examples" |

**When to Use Sonnet:**
- ✅ **Default choice for 80% of development tasks**
- ✅ When speed matters more than perfection
- ✅ When cost is a concern (early-stage startup)
- ✅ Iterative development (try, test, refine)

**Cost:**
- **API:** $3/M input, $15/M output (5x cheaper than Opus)
- **Pro Subscription:** $20/month
- **Estimated Monthly:** $50-100 for daily use

**Recommendation: Primary workhorse for daily coding**

---

### 1.3 Claude Haiku 3.5

**Capabilities:**
- **Intelligence:** Good (suitable for simple tasks)
- **Speed:** 10x faster than Opus
- **Cost:** $0.25/M input, $1.25/M output (60x cheaper than Opus!)
- **Strengths:** Fast responses, cheap, good for simple tasks

**Best Use Cases:**
- Quick code snippets
- Simple documentation
- Formatting/linting suggestions
- Bash scripts
- Config files (YAML, JSON)

**When to Use Haiku:**
- ✅ Speed > intelligence (latency-sensitive)
- ✅ Cost optimization (high-volume tasks)
- ✅ Simple, repetitive work

**Recommendation: Use for 10-20% of tasks (simple, high-volume)**

---

## 2. Google Gemini

### 2.1 Gemini 2.0 Flash

**Capabilities:**
- **Intelligence:** Very high (comparable to Sonnet)
- **Speed:** Extremely fast (2M tokens/sec output!)
- **Context:** 1M tokens (5x larger than Claude)
- **Cost:** **FREE** up to 1,500 requests/day (generous!)
- **Strengths:**
  - Fast code completion
  - Large context (can process entire codebase)
  - Multimodal (can analyze diagrams, screenshots)
  - Free tier is production-ready

**Best Use Cases for LuminetIQ:**

| Task | Why Gemini | Example |
|------|------------|---------|
| **Large Context Analysis** | 1M tokens = entire codebase | "Analyze all Go files in internal/ for common patterns" |
| **Documentation Generation** | Free + fast | "Generate API docs for all endpoints in one shot" |
| **Screenshot Analysis** | Multimodal | "Analyze this WiFi heatmap screenshot and suggest improvements" |
| **Code Search** | Fast semantic search | "Find all places we use unsafe pointer operations" |
| **Batch Processing** | Free tier = 1,500 req/day | "Generate tests for all 50 API endpoints" |

**When to Use Gemini:**
- ✅ Cost optimization (use free tier first)
- ✅ Very large context needed (>200K tokens)
- ✅ Image/diagram analysis
- ✅ Batch operations (leverage free tier)

**Cost:**
- **Free Tier:** 1,500 requests/day (!!!)
- **Paid:** $0.30/M input, $1.20/M output (cheaper than Claude)
- **Estimated Monthly:** $0-20 (mostly free tier)

**Recommendation: Primary tool for documentation and large-context analysis**

---

### 2.2 Gemini 2.0 Pro

**Capabilities:**
- **Intelligence:** Highest (competitive with Opus)
- **Context:** 2M tokens (10x Claude!)
- **Multimodal:** Vision, audio, video
- **Cost:** $1.25/M input, $10/M output (cheaper than Opus)

**Best Use Cases:**
- Processing entire codebase in one shot
- Video analysis (e.g., analyzing competitor product demos)
- Complex reasoning with massive context

**Recommendation: Use sparingly (Flash is usually sufficient)**

---

## 3. GitHub Copilot

**Capabilities:**
- **Real-time code completion** in VSCode/JetBrains/Vim
- **Context-aware:** Uses surrounding code
- **Multi-language:** Go, TypeScript, React, Python, etc.
- **Chat mode:** Ask questions in IDE

**Best Use Cases for LuminetIQ:**

| Task | Why Copilot | Example |
|------|-------------|---------|
| **Boilerplate Code** | Autocomplete reduces typing by 40% | Start typing "func (s *Scanner) Scan..." and it completes |
| **Repetitive Patterns** | Learns your coding style | JSON marshaling, error handling, struct definitions |
| **Test Generation** | Autocomplete test cases | Type "func TestDeviceClassification..." and it generates tests |
| **Documentation Comments** | Suggests doc comments | Type "// Package discovery..." and it completes |
| **Inline Chat** | Ask questions without leaving IDE | "Why is this function slow?" |

**When to Use Copilot:**
- ✅ **ALWAYS** - runs in background, minimal friction
- ✅ Writing new functions (autocomplete 40-60% of code)
- ✅ Exploring APIs (suggests method calls)
- ✅ Learning new libraries (shows examples)

**Cost:**
- **Individual:** $10/month
- **Business:** $19/user/month
- **Estimated Monthly:** $10-19

**Recommendation: ESSENTIAL TOOL - must-have for daily coding**

---

## 4. OpenAI ChatGPT

### 4.1 ChatGPT o1-preview / o1-pro

**Capabilities:**
- **Reasoning:** Extended "thinking" time (up to 60 seconds)
- **Strengths:**
  - Mathematical proofs
  - Algorithm design
  - Complex problem-solving
  - Physics/RF calculations

**Best Use Cases for LuminetIQ:**

| Task | Why o1 | Example |
|------|--------|---------|
| **RF Propagation Math** | Physics calculations, path loss formulas | "Derive log-distance path loss formula with wall attenuation" |
| **Algorithm Design** | Deep reasoning, optimization | "Design genetic algorithm for AP placement with constraints" |
| **Performance Analysis** | Big-O analysis, optimization | "Optimize this O(n²) device discovery to O(n log n)" |
| **Security Analysis** | Threat modeling, attack vectors | "Analyze this authentication flow for TOCTOU vulnerabilities" |

**When to Use o1:**
- ✅ Math-heavy problems (RF propagation, statistics)
- ✅ Algorithm design (optimization, search)
- ✅ Formal reasoning (proofs, security analysis)

**Cost:**
- **o1-preview:** $15/M input, $60/M output
- **o1-pro:** $200/month subscription (higher limits)
- **Estimated Monthly:** $20-50 (occasional use)

**Recommendation: Use for 5-10% of tasks (math, algorithms)**

---

### 4.2 ChatGPT GPT-4 Turbo

**Capabilities:**
- **Intelligence:** High (similar to Claude Sonnet)
- **Speed:** Fast
- **Cost:** $10/M input, $30/M output

**Best Use Cases:**
- General-purpose coding
- Brainstorming
- Quick Q&A

**When to Use GPT-4:**
- ✅ When you're already in ChatGPT interface
- ✅ Need image generation (DALL-E integration)
- ✅ Web search integration (ChatGPT can browse)

**Recommendation: Backup option (Claude Sonnet is better for code)**

---

## 5. AI Tool Decision Matrix

### By Task Type

| Task | Best Tool | Why | Cost/Task |
|------|-----------|-----|-----------|
| **System Architecture** | Claude Opus 4.5 | Deep reasoning, considers trade-offs | $1-5 |
| **Feature Implementation** | Claude Sonnet 4.5 | Fast, accurate, cost-effective | $0.10-0.50 |
| **Code Completion** | GitHub Copilot | Real-time, in-IDE, low friction | Included ($10/mo) |
| **Debugging** | Claude Sonnet 4.5 | Quick root cause, explains clearly | $0.05-0.20 |
| **Documentation** | Gemini 2.0 Flash | Free, fast, large context | $0 (free tier) |
| **Testing** | Claude Sonnet 4.5 or Gemini Flash | Generate comprehensive tests | $0-0.50 |
| **Math/Algorithms** | ChatGPT o1 | Extended reasoning, proofs | $0.50-2 |
| **Large Context Analysis** | Gemini 2.0 Flash | 1M tokens, free | $0 |
| **Code Review** | Claude Opus 4.5 | Catches subtle bugs, security | $1-3 |
| **Refactoring** | Claude Sonnet 4.5 | Safe transformations, maintains behavior | $0.20-1 |
| **Sales Copy** | Claude Opus 4.5 | Persuasive, nuanced, strategic | $0.50-2 |
| **Marketing Content** | ChatGPT GPT-4 | Creative, engaging, SEO-friendly | $0.10-0.50 |

---

### By Use Case (Specific to LuminetIQ)

| Use Case | Recommended Tool | Alternative | Notes |
|----------|------------------|-------------|-------|
| **Implement WiFi RF Path Loss Algorithm** | ChatGPT o1-pro | Claude Opus | Need heavy math, physics reasoning |
| **Build REST API Endpoints** | Claude Sonnet 4.5 | Gemini Flash | Boilerplate + business logic, fast iteration |
| **Write Go Unit Tests** | Claude Sonnet 4.5 | GitHub Copilot | Comprehensive test coverage, edge cases |
| **Generate API Documentation** | Gemini 2.0 Flash | Claude Sonnet | Large context (all endpoints at once), free |
| **Debug Performance Issue** | Claude Sonnet 4.5 | Claude Opus if complex | Quick profiling analysis, optimization suggestions |
| **Design AI Root Cause Analysis** | Claude Opus 4.5 | ChatGPT o1 | Complex multi-system reasoning |
| **Write Sales Playbook** | Claude Opus 4.5 | ChatGPT GPT-4 | Nuanced messaging, competitive positioning |
| **Create Marketing Blog Post** | ChatGPT GPT-4 | Claude Sonnet | SEO, engagement, creativity |
| **Review Security of Auth Code** | Claude Opus 4.5 | - | Catches subtle vulnerabilities |
| **Generate React Components** | GitHub Copilot | Claude Sonnet | Real-time autocomplete, fast iteration |

---

## 6. Recommended Workflow

### 6.1 Daily Development Workflow

**Morning (Planning & Architecture):**
1. **Claude Opus:** Review roadmap, prioritize tasks, architect complex features
2. **GitHub Copilot:** Keep running in background

**Daytime (Coding):**
1. **GitHub Copilot:** Autocomplete as you type (40% of code)
2. **Claude Sonnet:** When stuck, need debugging, or implementing features
3. **Gemini Flash:** Quick documentation lookups, large context searches

**Afternoon (Testing & Review):**
1. **Claude Sonnet:** Generate tests, review code
2. **Gemini Flash:** Batch generate docs for new features
3. **Claude Opus:** Final review before commit (complex features only)

**Evening (Documentation & Planning):**
1. **Gemini Flash:** Generate/update documentation
2. **ChatGPT GPT-4:** Write blog posts, marketing content
3. **Claude Opus:** Strategic planning for next day/week

---

### 6.2 Feature Development Workflow Example

**Feature:** Implement Predictive WiFi Survey

**Step 1: Architecture Design (Claude Opus)**
```
Prompt: "Design the architecture for predictive WiFi survey feature:
- RF path loss modeling (FSPL + log-distance + wall attenuation)
- Heatmap generation (IDW interpolation)
- AP placement optimization (genetic algorithm)
- Floor plan editor (React canvas)

Consider: performance (10K sample points), accuracy (±10 dB), extensibility."

Output: Detailed architecture doc, package structure, data flow
Cost: ~$3
Time: 20 minutes
```

**Step 2: Implement Core Algorithm (ChatGPT o1)**
```
Prompt: "Implement log-distance path loss formula with wall attenuation:
PL(d) = PL(d0) + 10×n×log10(d/d0) + Σ(wall_attenuation)

Include:
- FSPL for free-space reference
- Path loss exponent (n) by environment (office, hospital, warehouse)
- Wall materials (drywall 3dB, concrete 10dB, metal 20dB)
- Ray tracing for line-of-sight calculation"

Output: Go code with math, unit tests, validation
Cost: ~$2
Time: 30 minutes
```

**Step 3: Implement REST API (Claude Sonnet + Copilot)**
```
# Copilot autocompletes as you type:
type PredictiveSurveyRequest struct {
    FloorPlan FloorPlan `json:"floor_plan"`
    APLocations []Point `json:"ap_locations"`
    // Copilot suggests remaining fields...
}

# Claude Sonnet helps with business logic:
Prompt: "Implement POST /api/survey/predict endpoint that:
1. Validates floor plan (walls, dimensions)
2. Calls RF propagation model
3. Generates heatmap (100x100 grid, IDW interpolation)
4. Returns JSON with heatmap + coverage stats"

Output: Complete handler function with validation, tests
Cost: ~$0.50
Time: 15 minutes
```

**Step 4: Frontend Implementation (Claude Sonnet + Copilot)**
```
# Copilot autocompletes React components
# Claude Sonnet helps with complex state management:

Prompt: "Implement React component for floor plan editor:
- Canvas-based drawing (walls, doors, windows)
- Drag-and-drop AP placement
- Real-time heatmap overlay (fetched from API)
- Material selection (drywall, concrete, metal)
Use React hooks, TypeScript, Tailwind CSS"

Output: Complete React component with TypeScript types
Cost: ~$0.80
Time: 25 minutes
```

**Step 5: Testing (Claude Sonnet)**
```
Prompt: "Generate unit tests for RF path loss algorithm:
- Test FSPL calculation at 1m, 10m, 100m
- Test log-distance with different path loss exponents
- Test wall attenuation (1 wall, 5 walls, mixed materials)
- Test edge cases (d=0, negative attenuation, etc.)"

Output: Comprehensive Go tests with edge cases
Cost: ~$0.30
Time: 10 minutes
```

**Step 6: Documentation (Gemini Flash - FREE)**
```
Prompt: "Generate documentation for predictive WiFi survey:
- Package overview
- API endpoint specs (request/response examples)
- Algorithm explanation (FSPL, log-distance)
- Usage examples
- Performance characteristics
- Accuracy expectations (±10 dB)"

Output: Markdown docs, API specs, code examples
Cost: $0 (free tier)
Time: 5 minutes
```

**Step 7: Code Review (Claude Opus)**
```
Prompt: "Review this predictive WiFi survey implementation:
[paste all code: algorithm, API, frontend, tests]

Check for:
- Performance issues (O(n²) loops, memory leaks)
- Security (input validation, injection attacks)
- Accuracy (are formulas correct? any typos?)
- Edge cases (what if floor plan is invalid?)
- Best practices (error handling, logging)"

Output: Detailed review with suggestions
Cost: ~$5
Time: 15 minutes
```

**Total:**
- **Cost:** ~$12
- **Time:** 2 hours
- **Value:** Feature that would take 40 hours manually = **20x productivity boost**

---

## 7. Cost Optimization Strategies

### 7.1 Minimize Costs Without Sacrificing Quality

**Strategy 1: Use Free Tiers Aggressively**
- **Gemini Flash:** 1,500 requests/day FREE
- Use for: documentation, batch processing, large context analysis
- **Savings:** $50-100/month

**Strategy 2: Right-Size Model Selection**
- Don't use Opus for simple tasks (use Sonnet or Haiku)
- Don't use Sonnet for boilerplate (use Copilot or Gemini)
- **Savings:** $100-200/month

**Strategy 3: Batch Operations**
- Generate all tests at once (one API call) vs one-by-one
- Document all endpoints together (use Gemini's large context)
- **Savings:** $20-50/month

**Strategy 4: Cache Common Queries**
- Save architecture decisions, design patterns, code snippets locally
- Don't re-ask the same questions
- **Savings:** $10-30/month

**Strategy 5: Use Copilot for 80% of Typing**
- $10/month flat fee = unlimited autocomplete
- Reduces need for AI code generation by 40%
- **Savings:** $50-100/month (reduced Sonnet/Opus usage)

---

### 7.2 Monthly Budget Allocation

**Recommended Monthly Spend: $200-400**

| Tool | Subscription | Usage Cost | Total | % of Budget |
|------|--------------|------------|-------|-------------|
| **Claude Pro** | $20 | $50-100 | $70-120 | 25-35% |
| **GitHub Copilot** | $19 | - | $19 | 5-10% |
| **Gemini** | $0 | $0-20 | $0-20 | 0-5% |
| **ChatGPT Plus** | $20 | $20-50 | $40-70 | 10-20% |
| **ChatGPT o1-pro** (optional) | $200 | - | $200 | 40-50% |
| **Total** | **$59-259** | **$70-170** | **$129-429** | **100%** |

**Lean Startup Budget (< $100/month):**
- ✅ GitHub Copilot: $19 (essential)
- ✅ Claude Pro: $20 (for API credits)
- ✅ Gemini Flash: $0 (free tier)
- ❌ Skip ChatGPT Plus (use free tier)
- ❌ Skip o1-pro (use o1-preview on demand)
- **Total: $39/month + pay-as-you-go Sonnet/Opus usage**

**Growth Budget ($200-400/month):**
- ✅ All of above
- ✅ ChatGPT Plus or o1-pro
- ✅ Increased Claude API usage (more Opus for complex tasks)

---

## 8. Specific Recommendations for LuminetIQ

### 8.1 Development Phase (Now - v0.110.0)

**Primary Tools:**
1. **GitHub Copilot ($19/mo):** Always on, autocomplete 40% of code
2. **Claude Sonnet 4.5 ($20/mo Pro + usage):** Daily coding, debugging
3. **Gemini 2.0 Flash (FREE):** Documentation, large context analysis
4. **Claude Opus 4.5 (as needed):** Architecture, code review ($50-100/mo usage)

**Total: $90-150/month**

**Workflow:**
- Morning: Opus for architecture (15 min)
- Day: Sonnet + Copilot for coding (8 hours)
- Evening: Gemini for docs (30 min)
- Weekly: Opus for code review (1 hour Friday)

---

### 8.2 Sales & Marketing Phase (v0.110.0+)

**Additional Tools:**
1. **ChatGPT GPT-4 ($20/mo Plus):** Blog posts, social media, emails
2. **Claude Opus 4.5 (increased usage):** Sales playbook, case studies, ROI calculators

**Total: $120-200/month**

**Workflow:**
- Sales content: Opus (persuasive, strategic)
- Marketing content: GPT-4 (creative, SEO)
- Documentation: Gemini (free, fast)

---

### 8.3 Growth Phase (v1.0.0+)

**Full Stack:**
- GitHub Copilot: $19/mo × team size (4 engineers = $76/mo)
- Claude Pro: $20/mo × team size (or enterprise API)
- Gemini: Free tier (supplement with paid if needed)
- ChatGPT o1-pro: $200/mo (1 seat for algorithm work)

**Total: $300-500/month for 4-person team**

---

## 9. AI Coding Assistant Comparison

| Feature | Claude Opus | Claude Sonnet | Gemini Flash | Copilot | ChatGPT o1 |
|---------|-------------|---------------|--------------|---------|------------|
| **Intelligence** | ★★★★★ | ★★★★☆ | ★★★★☆ | ★★★☆☆ | ★★★★★ |
| **Speed** | ★★☆☆☆ | ★★★★☆ | ★★★★★ | ★★★★★ | ★☆☆☆☆ |
| **Cost** | $$$$$ | $$$ | FREE | $ | $$$$$ |
| **Context Size** | 200K | 200K | 1M | 8K | 128K |
| **Code Quality** | ★★★★★ | ★★★★★ | ★★★★☆ | ★★★☆☆ | ★★★★☆ |
| **Reasoning** | ★★★★★ | ★★★★☆ | ★★★★☆ | ★★☆☆☆ | ★★★★★ |
| **Real-time Autocomplete** | ❌ | ❌ | ❌ | ✅ | ❌ |
| **IDE Integration** | ❌ | ❌ | ❌ | ✅ | ❌ |
| **Best For** | Architecture, review | Daily coding | Docs, batch | Autocomplete | Math, algorithms |

---

## 10. Action Plan

### Week 1: Setup
- [ ] Subscribe to GitHub Copilot ($19/mo)
- [ ] Subscribe to Claude Pro ($20/mo)
- [ ] Sign up for Gemini (free tier)
- [ ] Test each tool with sample tasks

### Week 2: Workflow Integration
- [ ] Configure Copilot in VSCode/IDE
- [ ] Create Claude API wrapper scripts
- [ ] Set up prompt templates for common tasks
- [ ] Document which tool for which task (this doc!)

### Week 3: Optimize
- [ ] Track costs (which tool used how much)
- [ ] Identify over-usage (using Opus when Sonnet would work)
- [ ] Batch operations to reduce API calls
- [ ] Leverage Gemini free tier more

### Month 2+: Scale
- [ ] Train team on AI workflow (when hired)
- [ ] Create company prompt library
- [ ] Monitor ROI (time saved vs cost)
- [ ] Adjust tool mix as needs evolve

---

## 11. Conclusion

**Recommended Stack for LuminetIQ:**

**Core Tools (Essential):**
1. **GitHub Copilot ($19/mo):** Real-time autocomplete, 40% of code written
2. **Claude Sonnet 4.5 ($20/mo Pro + $50-100 usage):** Daily coding workhorse
3. **Gemini 2.0 Flash (FREE):** Documentation, large context

**Strategic Tools (High-Value):**
4. **Claude Opus 4.5 ($50-100/mo usage):** Architecture, code review, sales
5. **ChatGPT o1 ($20-200/mo):** Math, algorithms, RF calculations

**Total Monthly Cost: $129-439**
**Developer Productivity Gain: 5-20x**
**ROI: 10x minimum** (saves $2,000-8,000/month in developer time)

**Key Insight:**
- **Use the right tool for the job**
- Don't use a sledgehammer (Opus) when a screwdriver (Sonnet) will do
- Don't penny-pinch on tools ($200/mo) if it saves 40 hours/month ($4,000 value)

**Bottom Line:**
Invest $200-400/month in AI tools, get 10-20x productivity boost = **save $2K-8K/month in developer time** = **best investment you'll make**.

---

**Document Owner:** Engineering Team
**Next Review:** Monthly (optimize tool usage, track ROI)
