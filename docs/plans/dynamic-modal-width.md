# Dynamic Modal Width for Key View

## Overview

Currently, the key view modal has a fixed maximum width of 600px, which works well for short values but can make long values difficult to read as they wrap extensively. This plan implements dynamic width calculation for the modal based on content length, with sensible minimum and maximum constraints to maintain good UX across different value sizes.

**Key Benefits:**
- Better readability for long values
- Maintains compact size for short values
- Prevents modal from becoming too wide and unwieldy
- Improves overall user experience when viewing various key lengths

## Implementation Approach

Development will follow an iterative approach with discrete phases. Each step will be implemented fully before moving to the next, with tests run after each change to verify nothing breaks.

## Implementation Steps

### Iteration 1: CSS Foundation ✓
- [ ] Update modal CSS to use responsive width with constraints
- [ ] Define CSS custom properties for min/max modal widths
- [ ] Add transition for smooth width changes
- [ ] Test current modal behavior with new CSS

**Technical Details:**
```css
.modal {
    width: clamp(600px, var(--modal-width, 600px), 1200px);
    transition: width 0.2s ease-out;
}
```

### Iteration 2: Content Measurement
- [ ] Add JavaScript function to measure content width
- [ ] Calculate appropriate modal width based on:
  - Key length
  - Value length (considering wrapping)
  - Available viewport width
- [ ] Apply calculated width via CSS custom property
- [ ] Test with various content lengths

**Width Calculation Logic:**
- Minimum width: 600px (current default)
- Maximum width: min(1200px, 90vw) (responsive to viewport)
- Base calculation: max(key length, value first-line length) + padding + margins
- Consider font size and character width for estimation

### Iteration 3: Dynamic Width Application
- [ ] Hook into HTMX `htmx:afterSwap` event for view modal
- [ ] Measure content after modal content loads
- [ ] Set `--modal-width` CSS variable
- [ ] Handle binary/base64 encoded values appropriately
- [ ] Test with short, medium, and long values

**Implementation in app.js:**
```javascript
document.addEventListener('htmx:afterSwap', function(event) {
    if (event.detail.target.id === 'modal-content') {
        adjustModalWidth();
    }
});
```

### Iteration 4: Edge Cases & Refinement
- [ ] Handle very long single-line values (prevent horizontal scroll)
- [ ] Test with multi-line values
- [ ] Verify behavior with binary indicators
- [ ] Check responsive behavior on mobile/tablet
- [ ] Ensure edit/form modals maintain appropriate width
- [ ] Test theme switching doesn't break width calculation

**Edge Cases to Consider:**
- Values with no spaces (single long word)
- Multi-line JSON/YAML values
- Base64 encoded binary data
- Very short keys with very long values
- Mobile viewport constraints

### Iteration 5: Testing & Documentation
- [ ] Manual testing with various value types:
  - Short config values (< 50 chars)
  - Medium config values (50-200 chars)
  - Long JSON configs (> 200 chars)
  - Base64 binary data
- [ ] Test across browsers (Chrome, Firefox, Safari)
- [ ] Verify no regression in modal open/close behavior
- [ ] Update comments in CSS/JS if needed

## Technical Details

### Width Constraints
- **Minimum width**: 600px (maintains current compact default)
- **Maximum width**: 1200px or 90vw (whichever is smaller)
- **Default width**: Falls back to 600px if calculation fails

### Content Measurement Approach
1. Create temporary hidden measurement element
2. Clone value content into measurement element
3. Measure rendered width
4. Calculate modal width with padding/margins
5. Apply via CSS custom property
6. Clean up measurement element

### Files to Modify
- `app/server/static/style.css` - Modal width CSS
- `app/server/static/app.js` - Width calculation logic
- No template changes needed (existing structure works)

### Backward Compatibility
- Falls back to current 600px width if JavaScript fails
- Uses `clamp()` with fallback for older browsers
- Progressive enhancement approach - works without JS

## Success Criteria
- [ ] Short values (< 50 chars) use ~600px width
- [ ] Medium values (50-200 chars) scale appropriately
- [ ] Long values use maximum width without horizontal scroll
- [ ] Modal never exceeds viewport width
- [ ] Smooth transition when content changes
- [ ] No layout shift or flicker
- [ ] Works in all supported browsers
