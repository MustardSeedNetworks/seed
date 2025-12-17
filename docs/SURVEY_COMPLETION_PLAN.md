# Survey Feature Completion Plan

> Created: December 17, 2024 Epic: #401 - Implement a comprehensive WiFi Site Survey suite Goal:
> Complete professional-grade WiFi survey capabilities

---

## Current State

### What's Done ✅

| Feature              | Location                    | Notes                                       |
| -------------------- | --------------------------- | ------------------------------------------- |
| Survey CRUD          | `internal/survey/survey.go` | Create, Get, List, Delete                   |
| Survey lifecycle     | `internal/survey/survey.go` | Created → In Progress → Paused → Completed  |
| Floor plan upload    | `FloorPlan` struct          | Base64 image with scale (m/pixel)           |
| Passive scanning     | `PassiveSample`             | All visible networks at each point          |
| Active monitoring    | `ActiveSample`              | Connected AP RSSI, roaming detection        |
| Throughput testing   | `ThroughputSample`          | iPerf3 integration (up/down/latency/jitter) |
| Sample collection    | `AddSample()`               | X,Y coordinates tied to floor plan          |
| Band aggregation     | `CalculateAggregations()`   | 2.4/5/6 GHz AP counts per sample            |
| Channel interference | `CalculateAggregations()`   | Co-channel and adjacent AP counts           |
| AirMapper import     | `airmapper_parser.go`       | Parse NetAlly survey files                  |
| Persistence          | `saveSurvey()`              | JSON file storage                           |
| API endpoints        | `internal/api/`             | Full REST API for surveys                   |

### What's Missing ❌

| Category           | Feature                       | Issue      |
| ------------------ | ----------------------------- | ---------- |
| **Visualization**  | Heatmap rendering             | #646       |
| **Visualization**  | Channel overlap graph         | #167       |
| **Analysis**       | Dead zone detection           | #652, #589 |
| **Analysis**       | Channel interference analysis | #592       |
| **Analysis**       | Roaming pattern analysis      | #593       |
| **Export**         | PDF report generation         | #653, #595 |
| **Infrastructure** | Multi-floor support           | #654       |
| **Infrastructure** | Multi-adapter support         | #573       |
| **Infrastructure** | Survey MCP tools              | #650       |
| **Enhancement**    | Coverage estimator            | #406       |

---

## Implementation Phases

### Phase 1: Core Visualization (Foundation)

**Goal**: See survey data visually

| Task                          | Issue | Priority | Effort | Dependencies |
| ----------------------------- | ----- | -------- | ------ | ------------ |
| Heatmap rendering engine      | #646  | P0       | Large  | None         |
| Channel overlap visualization | #167  | P1       | Medium | None         |

#### Heatmap Engine Requirements (#646)

```
Input: Survey samples with RSSI values + floor plan
Output: PNG/SVG heatmap overlay

Algorithm:
1. Create grid over floor plan (resolution configurable)
2. For each grid cell, interpolate RSSI from nearby samples
3. Apply color gradient based on signal strength
4. Composite with floor plan image
```

**Visualization Types**:

- RSSI coverage (signal strength)
- SNR (signal-to-noise ratio)
- AP density (network count per area)
- Band utilization (2.4 vs 5 vs 6 GHz)
- Channel interference (co-channel AP count)

**Deliverables**:

- [ ] `internal/survey/heatmap.go` - Core rendering engine
- [ ] `internal/survey/interpolation.go` - IDW/kriging algorithms
- [ ] `internal/survey/colorscale.go` - Gradient definitions
- [ ] API endpoint: `GET /api/survey/{id}/heatmap?type=rssi`
- [ ] Unit tests with 80%+ coverage

---

### Phase 2: Analysis Features

**Goal**: Automatically find problems in survey data

| Task                          | Issue | Priority | Effort | Dependencies   |
| ----------------------------- | ----- | -------- | ------ | -------------- |
| Dead zone detection           | #652  | P1       | Medium | Heatmap (#646) |
| Channel interference analysis | #592  | P1       | Medium | None           |
| Roaming analysis              | #593  | P2       | Medium | Active surveys |

#### Dead Zone Detection (#652)

```
Input: Survey samples + RSSI threshold (default: -75 dBm)
Output: List of dead zones with locations and severity

Algorithm:
1. Find all samples below threshold
2. Cluster nearby weak samples into zones
3. Calculate zone center, radius, severity
4. Generate recommendations
```

**Deliverables**:

- [ ] `internal/survey/analysis.go` - Analysis functions
- [ ] `DeadZone` struct with location, severity, recommendations
- [ ] API endpoint: `GET /api/survey/{id}/dead-zones?threshold=-75`
- [ ] Integration with heatmap (highlight dead zones)

#### Channel Interference Analysis (#592)

```
Input: Passive survey samples
Output: Interference report with channel recommendations

Analysis:
1. Count APs per channel across all samples
2. Identify co-channel interference hotspots
3. Calculate channel utilization per band
4. Recommend optimal channel assignments
```

**Deliverables**:

- [ ] `internal/survey/channel_analysis.go`
- [ ] Channel utilization chart data
- [ ] Interference heatmap layer
- [ ] Channel recommendation engine

---

### Phase 3: Export & Reporting

**Goal**: Share survey results professionally

| Task                     | Issue | Priority | Effort | Dependencies      |
| ------------------------ | ----- | -------- | ------ | ----------------- |
| PDF report generation    | #653  | P2       | Large  | Heatmap, Analysis |
| Export to common formats | -     | P2       | Medium | None              |

#### Report Generation (#653)

**Report Sections**:

1. Executive Summary
   - Survey metadata (name, date, type)
   - Overall coverage score (0-100)
   - Key findings (3-5 bullets)

2. Floor Plan & Heatmap
   - Annotated floor plan with sample points
   - RSSI heatmap overlay
   - Dead zone markers

3. Coverage Analysis
   - Signal distribution chart
   - Coverage by quality level (excellent/good/fair/poor)
   - Band utilization breakdown

4. Interference Analysis
   - Channel utilization chart
   - Top interfering networks
   - Channel recommendations

5. Recommendations
   - Dead zone remediation
   - Channel optimization
   - Additional AP suggestions

**Deliverables**:

- [ ] `internal/survey/report.go` - Report generator
- [ ] HTML report using `html/template` (print to PDF via browser)
- [ ] API endpoint: `GET /api/survey/{id}/report` (returns HTML)
- [ ] Customizable branding (logo, colors via CSS)
- [ ] Embedded charts via Chart.js (inline script)
- [ ] Print-optimized CSS (`@media print`)

#### Export Formats

| Format | Use Case                      | Implementation  |
| ------ | ----------------------------- | --------------- |
| HTML   | View in browser, print to PDF | `html/template` |
| CSV    | Raw data for spreadsheets     | `encoding/csv`  |
| JSON   | API/integration               | `encoding/json` |

---

### Phase 4: Infrastructure Enhancements

**Goal**: Support complex survey scenarios

| Task                  | Issue | Priority | Effort | Dependencies |
| --------------------- | ----- | -------- | ------ | ------------ |
| Multi-floor surveys   | #654  | P2       | Large  | None         |
| Multi-adapter support | #573  | P2       | Medium | None         |
| Survey MCP tools      | #650  | P1       | Medium | All above    |

#### Multi-Floor Support (#654)

```go
type Building struct {
    ID     string   `json:"id"`
    Name   string   `json:"name"`
    Floors []*Floor `json:"floors"`
}

type Floor struct {
    ID        string     `json:"id"`
    Name      string     `json:"name"`      // "Floor 1", "Basement"
    Level     int        `json:"level"`     // Numeric level
    FloorPlan *FloorPlan `json:"floor_plan"`
    Samples   []*Sample  `json:"samples"`
}
```

**Deliverables**:

- [ ] Update `Survey` struct to support multiple floors
- [ ] Floor management API (add/remove/switch floors)
- [ ] Per-floor and building-wide statistics
- [ ] Floor navigation in UI

#### Multi-Adapter Support (#573)

**Use Case**: Use one adapter for scanning, another for connectivity

```yaml
wifi:
  survey:
    scan_interface: wlan0 # High-capability adapter
    upload_interface: wlan1 # Stay connected for uploads
```

**Deliverables**:

- [ ] Adapter selection in survey config
- [ ] Detect available WiFi adapters
- [ ] Automatic fallback if adapter unavailable

---

### Phase 5: MCP Integration

**Goal**: Enable AI to work with surveys

| Tool             | Description                             |
| ---------------- | --------------------------------------- |
| `survey_create`  | Create new survey                       |
| `survey_list`    | List all surveys                        |
| `survey_get`     | Get survey details                      |
| `survey_analyze` | Run analysis (dead zones, interference) |
| `survey_heatmap` | Generate heatmap image                  |
| `survey_report`  | Generate PDF report                     |

**Deliverables**:

- [ ] `internal/mcp/tools_survey.go` - Survey management tools
- [ ] `internal/mcp/tools_survey_analysis.go` - Analysis tools
- [ ] Tool documentation

---

## Dependency Graph

```
                          ┌─────────────────────┐
                          │   Heatmap Engine    │
                          │       #646          │
                          └──────────┬──────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
              ▼                      ▼                      ▼
   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
   │  Dead Zone      │    │  Channel        │    │  Channel        │
   │  Detection #652 │    │  Analysis #592  │    │  Overlap #167   │
   └────────┬────────┘    └────────┬────────┘    └─────────────────┘
            │                      │
            └──────────┬───────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │  Report Generation  │
            │       #653          │
            └──────────┬──────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │  Survey MCP Tools   │
            │       #650          │
            └─────────────────────┘

   Independent tracks:
   ┌─────────────────┐    ┌─────────────────┐
   │  Multi-Floor    │    │  Multi-Adapter  │
   │     #654        │    │     #573        │
   └─────────────────┘    └─────────────────┘
```

---

## Issue Consolidation

### Duplicate/Related Issues to Consolidate

| Keep          | Close/Reference | Reason                       |
| ------------- | --------------- | ---------------------------- |
| #652 (survey) | #589 (wifi-ai)  | Same feature, survey-focused |
| #653 (survey) | #595 (ai)       | Same feature, survey-focused |
| #646 (survey) | #588 (wifi-ai)  | Same feature, survey-focused |

**Recommendation**: Close the wifi-ai versions and reference the survey issues.

---

## Effort Estimates

| Phase                   | Issues           | Total Effort   |
| ----------------------- | ---------------- | -------------- |
| Phase 1: Visualization  | #646, #167       | Large + Medium |
| Phase 2: Analysis       | #652, #592, #593 | Medium × 3     |
| Phase 3: Export         | #653             | Large          |
| Phase 4: Infrastructure | #654, #573       | Large + Medium |
| Phase 5: MCP            | #650             | Medium         |

---

## Acceptance Criteria (Survey Complete)

Survey feature is complete when:

- [ ] **Visualization**: Heatmaps render from survey data
- [ ] **Analysis**: Dead zones auto-detected with recommendations
- [ ] **Analysis**: Channel interference analyzed with suggestions
- [ ] **Export**: PDF reports generated with all sections
- [ ] **Multi-floor**: Buildings with multiple floors supported
- [ ] **MCP**: AI can create, analyze, and report on surveys
- [ ] **Tests**: 80%+ code coverage on new code
- [ ] **Docs**: API documentation complete

---

## Files to Create/Modify

### New Files

| File                                  | Purpose                          |
| ------------------------------------- | -------------------------------- |
| `internal/survey/heatmap.go`          | Heatmap rendering engine         |
| `internal/survey/interpolation.go`    | Spatial interpolation algorithms |
| `internal/survey/colorscale.go`       | Color gradient definitions       |
| `internal/survey/analysis.go`         | Dead zone, coverage analysis     |
| `internal/survey/channel_analysis.go` | Channel interference analysis    |
| `internal/survey/report.go`           | PDF report generation            |
| `internal/survey/building.go`         | Multi-floor data structures      |
| `internal/mcp/tools_survey.go`        | Survey MCP tools                 |

### Modified Files

| File                              | Changes                                     |
| --------------------------------- | ------------------------------------------- |
| `internal/survey/survey.go`       | Add Building support, analysis methods      |
| `internal/api/handlers_survey.go` | New endpoints for heatmap, analysis, report |
| `internal/api/routes.go`          | Register new endpoints                      |

---

## Quick Reference: All Survey Issues

| Issue | Title                         | Priority | Phase |
| ----- | ----------------------------- | -------- | ----- |
| #646  | Heatmap rendering engine      | P0       | 1     |
| #167  | Channel overlap graph         | P1       | 1     |
| #652  | Dead zone detection           | P1       | 2     |
| #592  | Channel interference analysis | P1       | 2     |
| #593  | Roaming pattern analysis      | P2       | 2     |
| #653  | Report generation             | P2       | 3     |
| #654  | Multi-floor support           | P2       | 4     |
| #573  | Multi-adapter support         | P2       | 4     |
| #650  | Survey MCP tools              | P1       | 5     |

**Total**: 9 issues across 5 phases
