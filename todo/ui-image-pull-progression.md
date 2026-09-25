# UI Image Pull Progression & Status Feedback

Brainstorming and architectural design for detecting when Docker Compose is pulling an image during service start, tracking its progression, and delivering real-time feedback to the user in Mandok.

---

## 1. Problem Statement

In [`app/lib/compose/actions.go`](file:///home/yuri/work/projects/docker/mandok/app/lib/compose/actions.go), the function [`StartProject`](file:///home/yuri/work/projects/docker/mandok/app/lib/compose/actions.go#L9) starts a service or whole project by invoking Docker Compose's `apiClient.Up(...)`:

```go
func StartProject(projectDir string, forceRecreate bool, pull bool, services ...string) error {
    ctx := context.Background()
    project, err := LoadProject(ctx, projectDir)
    if err != nil {
        return err
    }
    ...
    apiClient := getAPI()
    err = apiClient.Up(ctx, project.Project, api.UpOptions{
        Create: api.CreateOptions{
            Build: &api.BuildOptions{
                Pull: pull,
            },
            Recreate:      recreatePolicy,
            Services:      services,
            RemoveOrphans: true,
            AssumeYes:     true,
            Inherit:       true,
        },
        Start: api.StartOptions{
            Services: services,
            Project:  project.Project,
            Wait:     true,
        },
    })
    ...
}
```

### Key Issues in Current Workflow:
1. **Synchronous HTTP Request**: Web handlers in [`web/handlers/ax/dashboard.go`](file:///home/yuri/work/projects/docker/mandok/web/handlers/ax/dashboard.go#L207) and [`web/handlers/hx/dashboard.go`](file:///home/yuri/work/projects/docker/mandok/web/handlers/hx/dashboard.go#L212) call `compose.StartProject(...)` synchronously on an HTTP request.
2. **Hidden Progress**: Inside Compose v2 (`pkg/compose/up.go`), `Up()` streams layer downloads and extractions to `s.stdinfo()`, which defaults to the server process's `os.Stderr` (terminal console / systemd journal).
3. **No UI Feedback**: When pulling large images (500MB - 3GB+), the request can block for minutes. The user only sees a spinning button or frozen page, risking HTTP gateway timeouts (504 Gateway Timeout on Traefik / Nginx) and leaving the user wondering if the system crashed.
4. **Implicit Pulls**: Even if `pull` is `false`, Compose checks whether the image exists locally. If it is missing, Compose implicitly pulls it without informing the caller.

---

## 2. Detecting Image Pull Requirements

To know if pulling will occur, check **proactively before `Up`**:

| Scenario | Condition | Will Pull? | Detection Strategy |
| :--- | :--- | :--- | :--- |
| **Explicit Pull** | `pull == true` | **Yes** | User initiated action with `pull=true`. |
| **First Run / Missing Image** | `pull == false` and image not in local store | **Yes** | Image inspection fails with `dockerclient.IsErrNotFound(err)`. |
| **Cached Image** | `pull == false` and image present locally | **No** | Local image inspection succeeds. |

### Proactive Inspection Helper

```go
// NeedsPull checks if a service image must be pulled from a remote registry
func NeedsPull(ctx context.Context, imageName string, forcePull bool) (bool, error) {
    if forcePull {
        return true, nil
    }
    if imageName == "" {
        return false, nil
    }
    _, _, err := _dockerClient.ImageInspectWithRaw(ctx, imageName)
    if err != nil && dockerclient.IsErrNotFound(err) {
        return true, nil // Missing locally: Compose will pull it!
    }
    return false, err
}
```

---

## 3. Capturing Pull Progression

```
                  ┌───────────────────────────────────────────────┐
                  │                 Docker Daemon                 │
                  └──────────────────────┬────────────────────────┘
                                         │ JSON Message Stream
                                         ▼
                 ┌─────────────────────────────────────────────────┐
                 │       Pull Progress Parser & Aggregator         │
                 │   - Layer IDs, Download % & Extract %           │
                 │   - Bytes: Current / Total                      │
                 │   - Overall aggregated %                        │
                 └───────────────┬─────────────────┬───────────────┘
                                 │                 │
                (Method A)       ▼                 ▼    (Method B)
             Server-Sent Events (SSE)       In-Memory State Store
             /ax/project/:id/events         ServiceStatus { State: "pulling" }
                                 │                 │
                                 ▼                 ▼
                         Browser Frontend (Alpine.js / HTMX)
```

Docker Engine streams progress as newline-delimited JSON messages (`jsonmessage.JSONMessage`):
- `ID`: Layer hash (e.g. `c0f70f698f48`)
- `Status`: Current state (`"Pulling fs layer"`, `"Downloading"`, `"Extracting"`, `"Pull complete"`, `"Already exists"`)
- `Progress`: `Current` bytes, `Total` bytes

### Approach A: Explicit Pre-Pull via Docker Engine API *(Recommended)*
Instead of letting `apiClient.Up` silently download images to stderr, pull missing/requested images explicitly first:

1. Call `_dockerClient.ImagePull(ctx, imageRef, image.PullOptions{...})`.
2. Parse the returned `io.ReadCloser` with `json.NewDecoder`.
3. Aggregate per-layer bytes into an overall progress percentage.
4. Pass progress to a callback / channel.
5. Once pulling completes, call `apiClient.Up(..., Pull: false)` which completes in milliseconds because the image is already present in the local daemon.

#### Data Model
```go
type PullProgress struct {
    Service    string  `json:"service"`
    Image      string  `json:"image"`
    Status     string  `json:"status"`   // "Downloading", "Extracting", "Complete"
    Current    int64   `json:"current"`  // Total downloaded bytes across layers
    Total      int64   `json:"total"`    // Total image size across layers
    Percent    float64 `json:"percent"`  // 0 - 100%
    Layer      string  `json:"layer"`    // Active layer ID
}
```

#### Progress Aggregator Implementation
```go
func PullImageWithProgress(ctx context.Context, imageName string, onProgress func(PullProgress)) error {
    out, err := _dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
    if err != nil {
        return err
    }
    defer out.Close()

    layers := make(map[string]struct{ Current, Total int64 })
    decoder := json.NewDecoder(out)

    for {
        var jm jsonmessage.JSONMessage
        if err := decoder.Decode(&jm); err != nil {
            if err == io.EOF {
                break
            }
            return err
        }

        if jm.ID != "" && jm.Progress != nil {
            layers[jm.ID] = struct{ Current, Total int64 }{
                Current: jm.Progress.Current,
                Total:   jm.Progress.Total,
            }

            var sumCurrent, sumTotal int64
            for _, l := range layers {
                sumCurrent += l.Current
                sumTotal += l.Total
            }

            var percent float64
            if sumTotal > 0 {
                percent = float64(sumCurrent) / float64(sumTotal) * 100.0
            }

            if onProgress != nil {
                onProgress(PullProgress{
                    Image:   imageName,
                    Status:  jm.Status,
                    Current: sumCurrent,
                    Total:   sumTotal,
                    Percent: percent,
                    Layer:   jm.ID,
                })
            }
        }
    }
    return nil
}
```

### Approach B: Intercept Compose v2's Progress Writer
Compose v2 already translates Docker messages into [`progress.Event`](file:///home/yuri/go/pkg/mod/github.com/docker/compose/v2@v2.40.3/pkg/progress/writer.go#L48). However, Compose's `apiClient.Up` hardwires output to `s.stdinfo()` (`cli.Err()`).
- Capturing this requires instantiating a custom `command.DockerCli` per operation, wrapping `WithErrorStream(...)` with a pipe, and setting `progress.Mode = progress.ModeJSON`.
- This approach is heavier and more brittle due to global state flags in the Compose package.

---

## 4. Communication Architecture: Server to Browser

### 1. Asynchronous Job Triggering
Avoid holding an open synchronous HTTP POST/GET for several minutes:
1. User clicks **Start** or **Pull**.
2. Handler registers an in-memory job:
   `serviceState[serviceName] = { State: "pulling", Progress: 0 }`.
3. Handler returns immediately with `202 Accepted` (or immediate Alpine Ajax trigger).
4. Operation runs in a background goroutine.

### 2. Real-Time Delivery via Existing SSE
Mandok already runs Server-Sent Events at `/ax/project/:project/events` ([`web/handlers/ax/events.go`](file:///home/yuri/work/projects/docker/mandok/web/handlers/ax/events.go)).

1. **Broadcast `pull-progress` Events**:
   ```http
   event: pull-progress
   data: {"service":"web","image":"postgres:16","status":"Downloading","percent":42.5,"current":124000000,"total":295000000}
   ```
2. **Broadcast `updateStatus` on Completion**:
   When the pull finishes and container starts:
   ```http
   event: updateStatus
   data: {}
   ```
   The frontend automatically refreshes the table row to `running`.

---

## 5. UI/UX Presentation Options

### Option 1: In-Place Table Status with Mini Progress Bar *(Recommended)*
In [`web/templates/axs/components/status.templ`](file:///home/yuri/work/projects/docker/mandok/web/templates/axs/components/status.templ#L211-L216), enhance the `State` column with an Alpine.js listener:

```html
<td class="px-6 py-2 whitespace-nowrap"
    x-data="{ pullPercent: 0, pullStatus: '' }"
    @pull-progress.window="
        if ($event.detail.service === '{ v.Name }') {
            pullPercent = $event.detail.percent;
            pullStatus = $event.detail.status;
        }
    ">
    
    <!-- When pulling -->
    <template x-if="pullPercent > 0 && pullPercent < 100">
        <div class="flex flex-col w-48">
            <div class="flex justify-between text-xs mb-1">
                <span class="text-blue-600 font-semibold flex items-center gap-1">
                    <svg class="animate-spin h-3 w-3 text-blue-600" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                    </svg>
                    <span x-text="pullStatus">Pulling</span>
                </span>
                <span class="text-gray-500 font-mono" x-text="`${Math.round(pullPercent)}%`"></span>
            </div>
            <div class="w-full bg-gray-200 rounded-full h-1.5 overflow-hidden">
                <div class="bg-blue-600 h-1.5 rounded-full transition-all duration-300"
                     :style="`width: ${pullPercent}%`"></div>
            </div>
        </div>
    </template>

    <!-- Normal container state -->
    <template x-if="pullPercent === 0 || pullPercent >= 100">
        <div class="flex flex-col">
            <span class="text-sm font-medium">{ v.State }</span>
            <span class="text-xs text-gray-500">{ v.Status }</span>
        </div>
    </template>
</td>
```

### Option 2: Corner Toast in `EventList` *(Quick Prototype)*
Mandok already renders floating notifications in [`web/templates/axs/components/event.templ`](file:///home/yuri/work/projects/docker/mandok/web/templates/axs/components/event.templ#L12).
- Push a persistent toast to the event list when pull begins.
- Update its percentage in place.
- Dismiss toast once `Started` event is received.

### Option 3: Terminal / Layer Drawer Modal *(Detailed Power-User View)*
- Open a drawer or modal similar to Portainer / Docker Desktop.
- Render individual layers with download and extract progress bars:
  ```text
  Pulling library/nginx:latest...
  c0f70f698f48: Downloading  [=============>      ]  15.2MB / 28.1MB
  b254580b06b8: Extracting   [===================>]   3.4MB /  3.4MB
  a1b2c3d4e5f6: Pull complete
  ```

---

## 6. Implementation Checklist & Next Steps

1. **Backend Helper** ([`app/lib/compose/actions.go`](file:///home/yuri/work/projects/docker/mandok/app/lib/compose/actions.go)):
   - Implement `NeedsPull` check.
   - Implement `PullImageWithProgress` with aggregated byte progress.
   - Update `StartProject` to run pull with progress callback before `apiClient.Up`.

2. **Event Hub / SSE** ([`web/handlers/ax/events.go`](file:///home/yuri/work/projects/docker/mandok/web/handlers/ax/events.go)):
   - Create an in-memory event bus or broadcast channel for project-scoped events.
   - Add `event: pull-progress` stream handling.

3. **Frontend Component** ([`web/templates/axs/components/status.templ`](file:///home/yuri/work/projects/docker/mandok/web/templates/axs/components/status.templ)):
   - Add Alpine.js listener `@pull-progress.window`.
   - Render mini progress bar during active pull operations.
