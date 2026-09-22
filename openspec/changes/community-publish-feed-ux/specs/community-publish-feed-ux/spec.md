## Purpose

Aligns community convention dialog, topic association page, feed image grid fill, and publish multi-image / video cover preview UX with the product reference UI while keeping default remote media URLs on publish.

## ADDED Requirements

### Requirement: Convention dialog matches reference UI

The community convention dialog SHALL match the reference UI structure: title「社区公约」, multi-paragraph body welcoming users and stating respect rules, a tappable「点击了解完整社区公约」link affordance, and a primary full-width「我知道了」button. It SHALL remain client-only and at most once per calendar day.

#### Scenario: First open of publish today
- **WHEN** the user opens the publish screen for the first time that calendar day
- **THEN** the convention dialog appears with the reference layout elements above

#### Scenario: Acknowledge once per day
- **WHEN** the user taps「我知道了」
- **THEN** the dialog dismisses and does not appear again until the next calendar day

### Requirement: Feed image grid cells are fully filled

For image-type posts in the community feed, each grid cell SHALL display its image covering the entire cell (no large empty bands above or below the image within the cell). Images and video remain mutually exclusive; a video post SHALL show at most one video card.

#### Scenario: Nine-grid cell fill
- **WHEN** an image post is shown in the feed
- **THEN** each visible grid tile is filled edge-to-edge by its image (cover fit within a square cell)

#### Scenario: Video exclusive
- **WHEN** a post has video media
- **THEN** the feed shows one video card and does not show an image grid for that post

### Requirement: Topic association page matches reference UI

The topic association screen SHALL include: a search field with placeholder「搜索话题」; a highlighted top card for「问大家」with supporting description; and a scrollable topic list where each row shows `#` + topic name and a heat/popularity label on the right.

#### Scenario: Open topic page
- **WHEN** the user opens associate-topic from publish
- **THEN** they see search, the ask-everyone card (when the topic exists), and the topic list with heat on the right

#### Scenario: Search topics
- **WHEN** the user enters a query in the search field
- **THEN** the list filters to matching topics without leaving the screen chrome of the reference layout

### Requirement: Multi-select images up to nine with continue-add

On publish, choosing images SHALL allow selecting multiple images, up to 9 total. After selecting fewer than 9 (e.g. 4), the user SHALL be able to add more selections until the limit is reached. Exceeding 9 SHALL be prevented or truncated with feedback.

#### Scenario: Select four then add more
- **WHEN** the user has already selected 4 images and opens image pick again
- **THEN** they can add additional images so the draft holds more than 4 and at most 9

#### Scenario: Cap at nine
- **WHEN** the user attempts to select more than 9 images in total
- **THEN** the draft keeps at most 9 and the user is informed of the limit

### Requirement: Switching image and video clears the other

Changing the draft from image to video or from video to image SHALL clear the previous media selection including local previews and any video cover.

#### Scenario: Image to video
- **WHEN** the draft has selected images and the user successfully selects a video
- **THEN** all image selections and image previews are cleared and only the video draft remains

#### Scenario: Video to image
- **WHEN** the draft has a video (and optional cover) and the user successfully selects images
- **THEN** the video and cover are cleared and only the image draft remains

### Requirement: Video cover and media preview

After selecting a video, the user SHALL be able to set or see a cover image for the video and open a preview of the video. After selecting images, the user SHALL be able to tap an image to open an enlarged preview.

#### Scenario: Video cover and preview
- **WHEN** the user has selected a video on publish
- **THEN** a cover affordance is available and a preview action can open video playback

#### Scenario: Image enlarge preview
- **WHEN** the user has one or more selected images on publish and taps a thumbnail
- **THEN** an enlarged image preview opens for the selected set
