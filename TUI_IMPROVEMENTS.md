# TUI Selection Highlighting Improvements

## Changes Made

Enhanced the TUI to provide much better visual feedback for the currently selected line when navigating with arrow keys.

### Before
- Selected items had a subtle dark gray background (color 240)
- No visual indicator on the left
- Could be hard to see which line was selected

### After
- **Bright highlight background** (color 62 - purple/blue) with white text
- **Selection indicator** (❯) on the left of selected lines
- **Consistent highlighting** across all three views (Changes, Errors, Warnings)
- **Aligned spacing** - unselected lines have 2-space padding to align with the indicator

## Visual Examples

### Changes View

**Unselected lines:**
```
  ▸ ✚ aws_instance.web
  ▸ ~ aws_security_group.allow_http
  ▸ ✖ aws_s3_bucket.old_bucket
```

**Selected line (highlighted with purple background):**
```
  ▸ ✚ aws_instance.web
❯ ▸ ~ aws_security_group.allow_http  ← [HIGHLIGHTED IN PURPLE]
  ▸ ✖ aws_s3_bucket.old_bucket
```

### Errors View

**Before:**
```
> ✖ [resource] Error message
  ✖ Another error
  ✖ Yet another error
```

**After (selected line with full highlight):**
```
❯ ✖ [resource] Error message  ← [HIGHLIGHTED IN PURPLE]
  ✖ Another error
  ✖ Yet another error
```

### Warnings View

**Before:**
```
> ⚠ [resource] Warning message
  ⚠ Another warning
  ⚠ Yet another warning
```

**After (selected line with full highlight):**
```
❯ ⚠ [resource] Warning message  ← [HIGHLIGHTED IN PURPLE]
  ⚠ Another warning
  ⚠ Yet another warning
```

## Technical Details

### Style Changes

Updated `selectedStyle` from:
```go
selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("240")).Bold(true)
```

To:
```go
selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("15")).Bold(true)
```

This provides:
- Color 62: A nice purple/blue background (same as active tabs)
- Color 15: White foreground text for maximum contrast
- Bold text for additional emphasis

### Selection Indicator

Added `❯` (Unicode: U+276F) as a visual indicator:
- Appears on the left of selected lines
- 2-space padding on unselected lines for alignment
- Consistent across all view modes

### Code Changes

Modified three functions in `internal/tui/tui.go`:

1. **`renderTreeNode`** - Tree view selection (lines ~294-330)
2. **`renderErrorsView`** - Errors tab selection (lines ~469-488)
3. **`renderWarningsView`** - Warnings tab selection (lines ~490-509)

## User Experience Improvements

1. **Easier Navigation**: Users can now clearly see which resource they're about to expand
2. **Better Scanning**: The highlight makes it easy to track position while scrolling
3. **Consistent UX**: All three tabs use the same selection style
4. **Accessible**: High contrast (white on purple) improves readability

## Keyboard Navigation Reminder

The selection highlight moves with these keys:
- `↑` / `k` - Move up
- `↓` / `j` - Move down
- `g` - Jump to top
- `G` - Jump to bottom
- `Enter` / `Space` - Expand/collapse (Changes view only)

## Color Palette

The TUI now uses a cohesive color scheme:
- **Purple/Blue (62)**: Active tab background, selection highlight
- **White (15)**: Selected text
- **Green (10)**: Create actions
- **Yellow (11)**: Update actions
- **Red (9)**: Delete actions
- **Blue (12)**: Replace actions
- **Gray tones**: UI elements and borders

## Testing

To see the improvements:

```bash
# Build and run
go build ./cmd/tplan
terraform plan | ./tplan

# Navigate with arrow keys or j/k
# Notice the purple highlight that follows your selection
```

## Future Enhancements

Potential future improvements:
- Configurable colors via config file
- Theme support (light/dark modes)
- Custom selection indicator character
- Highlight animation on selection change

