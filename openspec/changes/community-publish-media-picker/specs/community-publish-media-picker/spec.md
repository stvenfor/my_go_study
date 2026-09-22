## Purpose

Defines how the community publish flow obtains image or video from the device gallery or camera for local preview, while the backend still stores only default media URLs.

## ADDED Requirements

### Requirement: Source chooser for image and video

When the user chooses image or video on the publish screen, the system SHALL present a source chooser that offers at least album (gallery) and camera options appropriate to that media kind.

#### Scenario: Tap image opens image sources
- **WHEN** the user taps the image media control on the publish screen
- **THEN** the system shows a chooser with album and camera (photo capture) options

#### Scenario: Tap video opens video sources
- **WHEN** the user taps the video media control on the publish screen
- **THEN** the system shows a chooser with album (video library) and camera (video capture) options

### Requirement: Real device media selection

The system SHALL open the platform gallery or camera so the user can actually pick or capture media; a successful pick MUST yield a local media reference available for preview on the publish screen.

#### Scenario: Pick image from album
- **WHEN** the user selects album for image and completes a successful pick
- **THEN** the publish screen shows a local image preview and the draft media type is image

#### Scenario: Capture photo with camera
- **WHEN** the user selects camera for image and completes a successful capture
- **THEN** the publish screen shows a local image preview and the draft media type is image

#### Scenario: Pick or capture video
- **WHEN** the user selects album or camera for video and completes a successful pick or capture
- **THEN** the publish screen indicates a selected video (preview or clear placeholder) and the draft media type is video

### Requirement: Image and video remain mutually exclusive

A post draft SHALL NOT hold both image and video at the same time. Selecting the other kind MUST replace the previous selection.

#### Scenario: Switch from image to video
- **WHEN** the draft already has an image selection and the user successfully selects a video
- **THEN** the image preview is cleared and only the video selection remains

### Requirement: Clear or cancel without forcing media

The user SHALL be able to cancel the chooser or clear the selected media so the draft returns to no media. Canceling mid-picker MUST NOT change the previous draft media state.

#### Scenario: Cancel picker
- **WHEN** the user opens a source and dismisses without selecting media
- **THEN** the draft media type and preview remain unchanged

#### Scenario: Clear media
- **WHEN** the user clears the selected media on the publish screen
- **THEN** the draft media type is none and local preview is removed

### Requirement: Publish still uses default remote URLs

Publishing MUST NOT upload the selected local file to the backend. The create-post request MUST continue to send `media_type` as `image` or `video` (and existing fields); the backend MUST continue to persist the agreed default remote image or video URLs.

#### Scenario: Publish after picking image
- **WHEN** the user has a local image preview and taps publish successfully
- **THEN** the created post in the feed shows the backend default image URL(s), not a device file path

#### Scenario: Publish after picking video
- **WHEN** the user has a local video selection and taps publish successfully
- **THEN** the created post in the feed shows the backend default video URL and cover, not a device file path

### Requirement: Permission denial is handled safely

If album or camera permission is denied, the system SHALL inform the user and MUST NOT crash or leave the publish screen in an inconsistent state.

#### Scenario: Permission denied
- **WHEN** the user chooses a source and the platform denies permission
- **THEN** the system shows an understandable message and the draft media state is unchanged (or safely cleared if a partial pick failed)
