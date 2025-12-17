# AI, MCP, and Survey Integration Plan

> Generated: December 2024 Updated: December 17, 2024 Purpose: Strategic roadmap for integrating AI
> capabilities with MCP tools and WiFi survey/planner features

## Executive Summary

This document outlines the integration strategy for combining:

- **MCP Server** (22 network diagnostic tools)
- **AI Features** (24 planned issues #575-598)
- **Survey/Planning** (existing + competitor-inspired features)

The goal: Create an AI-powered network diagnostic and WiFi planning platform that rivals Ekahau,
Hamina, and NetAlly.

---

## Part 1: Current State Analysis

### What We Have

#### Survey Package (`internal/survey/`)

| Feature              | Status  | Notes                                |
| -------------------- | ------- | ------------------------------------ |
| Survey CRUD          | ✅ Done | Create, read, update, delete surveys |
| Floor plan upload    | ✅ Done | Base64 image storage                 |
| Passive scanning     | ✅ Done | All visible networks                 |
| Active monitoring    | ✅ Done | Current connection RSSI              |
| Throughput testing   | ✅ Done | iPerf3 integration                   |
| Sample points        | ✅ Done | X,Y coordinates + data               |
| AirMapper import     | ✅ Done | NetAlly file parsing                 |
| Band aggregation     | ✅ Done | 2.4/5/6 GHz AP counts                |
| Channel interference | ✅ Done | Co-channel/adjacent counting         |

#### MCP Tools (Survey-Related)

| Tool        | Status  | Notes                    |
| ----------- | ------- | ------------------------ |
| `wifi_scan` | ✅ Done | Returns visible networks |
| `wifi_info` | ✅ Done | Current connection info  |

### What's Missing (vs. Competitors)

Based on [Ekahau AI Pro](https://www.peerspot.com/products/ekahau-ai-pro-reviews),
[Hamina](https://hamina.com), and [NetAlly AirMapper](https://netally.com):

| Feature                   | Ekahau | Hamina | NetAlly | We Have | Priority |
| ------------------------- | ------ | ------ | ------- | ------- | -------- |
| **AI Auto-Planner**       | ✅     | ✅     | ❌      | ❌      | Critical |
| Heatmap generation        | ✅     | ✅     | ✅      | ❌      | Critical |
| Wall attenuation modeling | ✅     | ✅     | ❌      | ❌      | High     |
| 3D multi-floor planning   | ✅     | ✅     | ❌      | ❌      | High     |
| 6 GHz support             | ✅     | ✅     | ✅      | Partial | High     |
| AP placement optimization | ✅     | ✅     | ❌      | ❌      | Critical |
| RF propagation modeling   | ✅     | ✅     | ❌      | ❌      | High     |
| Channel planning          | ✅     | ✅     | ❌      | ❌      | High     |
| Capacity planning         | ✅     | ✅     | ❌      | ❌      | Medium   |
| Roaming analysis          | ✅     | ❌     | ❌      | ❌      | Medium   |
| Dead zone detection       | ✅     | ✅     | ✅      | ❌      | High     |
| BOM generation            | ✅     | ✅     | ❌      | ❌      | Medium   |
| Vendor AP database        | ✅     | ✅     | ✅      | ❌      | High     |
| Report generation         | ✅     | ✅     | ✅      | ❌      | Medium   |
| Cloud collaboration       | ❌     | ✅     | ✅      | ❌      | Low      |
| Spectrum analysis         | ✅     | ❌     | ✅      | ❌      | Medium   |
| Hybrid surveys            | ✅     | ✅     | ✅      | Partial | Done     |

---

## Part 2: Gap Analysis - GitHub Issues Status

### WiFi Planner Foundation - Epic #651

| Issue | Description                | Priority    | Status  |
| ----- | -------------------------- | ----------- | ------- |
| #646  | Heatmap rendering engine   | P0 Critical | Created |
| #647  | Wall and obstacle modeling | P0 Critical | Created |
| #648  | RF propagation calculator  | P0 Critical | Created |
| #649  | AP vendor database         | P1 High     | Created |
| #650  | Survey MCP tools           | P1 High     | Created |

### Still Missing (Future Issues)

| Feature                       | Description                                | Priority |
| ----------------------------- | ------------------------------------------ | -------- |
| Multi-floor support           | Link floors, model inter-floor propagation | High     |
| Survey export formats         | Export to PDF, Ekahau, Hamina formats      | Medium   |
| BOM generator                 | Generate equipment list from design        | Medium   |
| Spectrum analyzer integration | Visualize RF interference                  | Medium   |

### Survey MCP Tools (#650)

| Tool                  | Description                  | AI Use Case            |
| --------------------- | ---------------------------- | ---------------------- |
| `survey_create`       | Create new survey via MCP    | AI initiates surveys   |
| `survey_analyze`      | Analyze survey data          | AI interprets results  |
| `survey_heatmap`      | Generate heatmap from survey | AI visualizes coverage |
| `survey_recommend_ap` | Get AP placement suggestions | AI-driven design       |
| `survey_dead_zones`   | Find coverage gaps           | AI troubleshooting     |
| `survey_channel_plan` | Optimize channel assignments | AI optimization        |

---

## Part 3: Integration Architecture

### How AI + MCP + Survey Work Together

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Request                              │
│         "Design WiFi for this floor plan"                       │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AI Agent Orchestrator (#644)                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │   Planner   │  │  Executor   │  │       Reasoner          │ │
│  │             │  │             │  │  (Interprets results,   │ │
│  │  Decides:   │  │  Calls MCP  │  │   generates report)     │ │
│  │  1. Analyze │  │   tools     │  │                         │ │
│  │  2. Design  │  │             │  │                         │ │
│  │  3. Optimize│  │             │  │                         │ │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘ │
└─────────┼────────────────┼─────────────────────┼───────────────┘
          │                │                     │
          ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                     MCP Server (seed mcp)                        │
│                                                                  │
│  Survey Tools          Network Tools         Analysis Tools      │
│  ─────────────         ─────────────         ──────────────      │
│  survey_create         wifi_scan             get_interfaces      │
│  survey_analyze        wifi_info             network_scan        │
│  survey_heatmap        get_neighbors         dns_test            │
│  survey_recommend_ap   traceroute            speedtest           │
│  survey_dead_zones     port_scan             vulnerability_scan  │
│  survey_channel_plan                                             │
└─────────────────────────────────────────────────────────────────┘
          │                │                     │
          ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Backend Services                              │
│                                                                  │
│  Survey Manager        WiFi Scanner         Network Manager      │
│  RF Propagation        Channel Planner      Device Discovery     │
│  Heatmap Generator     AP Database          Health Scoring       │
└─────────────────────────────────────────────────────────────────┘
```

### AI-Powered WiFi Planning Flow

```
Step 1: Input
─────────────────────────────────────────────────────────
User uploads floor plan → AI asks about:
  - Building type (office, warehouse, hospital)
  - Expected device density
  - Coverage requirements (voice, data, IoT)
  - Budget constraints

Step 2: Analysis (MCP Tools)
─────────────────────────────────────────────────────────
AI calls: wifi_scan → Detect existing networks
AI calls: survey_analyze → If existing survey data
AI calls: get_interfaces → Check available radios

Step 3: Design (AI + RF Model)
─────────────────────────────────────────────────────────
AI uses RF propagation model to:
  - Calculate coverage from potential AP locations
  - Account for wall attenuation
  - Ensure overlap for roaming
  - Avoid co-channel interference

AI calls: survey_recommend_ap → Get optimized placements

Step 4: Validation
─────────────────────────────────────────────────────────
AI calls: survey_heatmap → Generate predicted coverage
AI calls: survey_dead_zones → Identify gaps
AI calls: survey_channel_plan → Optimize channels

Step 5: Output
─────────────────────────────────────────────────────────
AI generates:
  - AP placement map
  - Predicted coverage heatmap
  - Channel plan
  - Bill of materials
  - Installation notes
```

---

## Part 4: Implementation Roadmap

### Phase 1: WiFi Planner Foundation - Epic #651

**Goal**: Build core planner infrastructure that AI can leverage

| Task                       | Issue | Dependencies   | Priority |
| -------------------------- | ----- | -------------- | -------- |
| Heatmap rendering engine   | #646  | None           | P0       |
| Wall/obstacle data model   | #647  | None           | P0       |
| RF propagation calculator  | #648  | Wall model     | P0       |
| AP database (JSON catalog) | #649  | None           | P1       |
| Survey MCP tools           | #650  | Heatmap engine | P1       |

### Phase 2: AI Integration (Weeks 5-8)

**Goal**: Connect AI agent to survey tools

| Task                    | Issue | Dependencies | Priority |
| ----------------------- | ----- | ------------ | -------- |
| AI service architecture | #575  | None         | P0       |
| AI agent orchestrator   | #644  | #575, MCP    | P0       |
| WiFi heatmap generation | #588  | Phase 1      | P1       |
| Dead zone detection     | #589  | Heatmap      | P1       |
| AP placement algorithm  | #590  | RF model     | P1       |

### Phase 3: Advanced Features (Weeks 9-12)

**Goal**: Match competitor capabilities

| Task                         | Issue    | Dependencies       | Priority |
| ---------------------------- | -------- | ------------------ | -------- |
| Predictive survey simulation | #591     | AI agent, RF model | P0       |
| Channel optimization         | #592     | AP database        | P1       |
| Roaming analysis             | #593     | Survey data        | P2       |
| Multi-floor support          | NEW #TBD | Wall model         | P1       |
| Report generation            | #595     | All above          | P2       |

### Phase 4: Differentiation (Weeks 13+)

**Goal**: Exceed competitors with AI-native features

| Task                        | Issue | Dependencies   | Priority |
| --------------------------- | ----- | -------------- | -------- |
| Natural language planning   | #585  | AI agent       | P1       |
| Predictive maintenance      | #594  | Baselines      | P2       |
| Root cause analysis         | #582  | AI agent       | P1       |
| Real-time anomaly detection | #583  | Time-series DB | P2       |

---

## Part 5: Issue Tracking

### Created Issues ✅

| Issue | Title                                    | Status  |
| ----- | ---------------------------------------- | ------- |
| #646  | feat(survey): Heatmap rendering engine   | Created |
| #647  | feat(survey): Wall and obstacle modeling | Created |
| #648  | feat(survey): RF propagation calculator  | Created |
| #649  | feat(survey): AP vendor database         | Created |
| #650  | feat(mcp): Survey MCP tools              | Created |
| #651  | epic: WiFi Planner Foundation (Phase 1)  | Created |

### Still To Create

6. **`feat(planner): Multi-floor building support`**
   - Link floors in a building
   - Inter-floor signal propagation
   - Stairwell/elevator shaft modeling

7. **`feat(planner): Survey report generator`**
   - PDF export with heatmaps
   - Executive summary
   - Recommendations section

8. **`epic: Survey/Planning Phase 1`**
   - Group foundational survey issues
   - Define dependencies
   - Track progress

---

## Part 6: Mapping Existing Issues to Plan

### Issues Ready to Implement (After Phase 1)

| Issue                             | Depends On         | Notes                |
| --------------------------------- | ------------------ | -------------------- |
| #588 WiFi heatmap generation      | Heatmap engine     | Core visualization   |
| #589 Dead zone detection          | Heatmap + analysis | Uses heatmap data    |
| #590 AP placement optimization    | RF model + AP DB   | AI-driven placement  |
| #591 Predictive survey simulation | All of Phase 1     | Flagship feature     |
| #592 Channel optimization         | AP database        | Automated planning   |
| #593 Roaming analysis             | Survey data        | Post-survey analysis |

### Issues That Enable Survey AI

| Issue                    | How It Helps Survey            |
| ------------------------ | ------------------------------ |
| #575 AI architecture     | Foundation for all AI features |
| #576 Time-series storage | Store historical survey data   |
| #577 Baseline learning   | Learn "normal" RF patterns     |
| #579 Health scoring      | Score network designs          |
| #585 NLP interface       | "Design WiFi for 50 users"     |
| #644 AI agent            | Orchestrates survey tools      |

---

## Part 7: Competitive Differentiation

### What We Can Do Better Than Ekahau/Hamina

| Differentiator             | How                                         |
| -------------------------- | ------------------------------------------- |
| **AI-Native**              | Built from ground up with AI, not bolted on |
| **Open Protocol**          | MCP allows any AI to use our tools          |
| **Real-time Analysis**     | Live network monitoring + survey            |
| **Integrated Diagnostics** | Survey + troubleshooting in one tool        |
| **Cost**                   | Open source core vs $3K+/year               |
| **Extensible**             | Custom tools via MCP                        |

### Unique Value Propositions

1. **"Survey While You Troubleshoot"**
   - Combine site survey with live diagnostics
   - AI correlates survey data with real-time issues

2. **"AI That Learns Your Network"**
   - Baseline learning from historical data
   - Recommendations improve over time

3. **"Natural Language WiFi Design"**
   - "Design WiFi for a 10,000 sqft warehouse with 50 IoT devices"
   - AI handles the rest

4. **"Predict Before You Deploy"**
   - Simulate network before installation
   - Validate against real measurements

---

## Part 8: Success Metrics

### Phase 1 Complete When:

- [ ] Heatmap renders from survey data
- [ ] Walls can be drawn on floor plans
- [ ] RF propagation calculates signal at any point
- [ ] 100+ APs in vendor database
- [ ] Survey MCP tools functional

### Phase 2 Complete When:

- [ ] AI agent can analyze survey data
- [ ] AI can recommend AP placements
- [ ] Dead zones automatically detected
- [ ] Channel plan auto-generated

### Phase 3 Complete When:

- [ ] Predictive survey matches real survey within 5dB
- [ ] Multi-floor designs supported
- [ ] Reports exportable to PDF
- [ ] Feature parity with Hamina free tier

### Phase 4 Complete When:

- [ ] NLP interface for WiFi design
- [ ] Real-time anomaly detection active
- [ ] Root cause analysis for WiFi issues
- [ ] Demonstrably better than Ekahau for specific use cases

---

## Appendix: Reference Links

### Competitor Research

- [Ekahau AI Pro Reviews](https://www.peerspot.com/products/ekahau-ai-pro-reviews)
- [Ekahau Features](https://www.ekahau.com)
- [Hamina Network Planner](https://hamina.com)
- [NetAlly AirMapper](https://www.netally.com)

### Our Issues

- [AI Foundation Epic #645](https://github.com/krisarmstrong/seed/issues/645)
- [MCP Tool Registry #640](https://github.com/krisarmstrong/seed/issues/640)
- [AI Agent Orchestrator #644](https://github.com/krisarmstrong/seed/issues/644)
- [WiFi Heatmap #588](https://github.com/krisarmstrong/seed/issues/588)
- [Predictive Survey #591](https://github.com/krisarmstrong/seed/issues/591)

### Technical References

- [MCP Protocol](https://modelcontextprotocol.io)
- [RF Propagation Models](https://en.wikipedia.org/wiki/Log-distance_path_loss_model)
- [IEEE 802.11 Standards](https://www.ieee802.org/11/)
