// SPDX-License-Identifier: Unlicense OR MIT

// +build darwin,!ios

@import AppKit;

#include "_cgo_export.h"

@interface GioAppDelegate : NSObject<NSApplicationDelegate>
@end

@interface GioWindowDelegate : NSObject<NSWindowDelegate>
@end

@implementation GioWindowDelegate
- (void)windowWillMiniaturize:(NSNotification *)notification {
	NSWindow *window = (NSWindow *)[notification object];
	gio_onHide((__bridge CFTypeRef)window.contentView);
}
- (void)windowDidDeminiaturize:(NSNotification *)notification {
	NSWindow *window = (NSWindow *)[notification object];
	gio_onShow((__bridge CFTypeRef)window.contentView);
}
- (void)windowDidChangeScreen:(NSNotification *)notification {
	NSWindow *window = (NSWindow *)[notification object];
	CGDirectDisplayID dispID = [[[window screen] deviceDescription][@"NSScreenNumber"] unsignedIntValue];
	CFTypeRef view = (__bridge CFTypeRef)window.contentView;
	gio_onChangeScreen(view, dispID);
}
- (void)windowDidBecomeKey:(NSNotification *)notification {
	NSWindow *window = (NSWindow *)[notification object];
	gio_onFocus((__bridge CFTypeRef)window.contentView, 1);
}
- (void)windowDidResignKey:(NSNotification *)notification {
	NSWindow *window = (NSWindow *)[notification object];
	gio_onFocus((__bridge CFTypeRef)window.contentView, 0);
}
- (void)windowWillClose:(NSNotification *)notification {
	NSWindow *window = (NSWindow *)[notification object];
	window.delegate = nil;
	gio_onClose((__bridge CFTypeRef)window.contentView);
}
@end

// Delegates are weakly referenced from their peers. Nothing
// else holds a strong reference to our window delegate, so
// keep a single global reference instead.
static GioWindowDelegate *globalWindowDel;

void gio_writeClipboard(unichar *chars, NSUInteger length) {
	@autoreleasepool {
		NSString *s = [NSString string];
		if (length > 0) {
			s = [NSString stringWithCharacters:chars length:length];
		}
		NSPasteboard *p = NSPasteboard.generalPasteboard;
		[p declareTypes:@[NSPasteboardTypeString] owner:nil];
		[p setString:s forType:NSPasteboardTypeString];
	}
}

CFTypeRef gio_readClipboard(void) {
	@autoreleasepool {
		NSPasteboard *p = NSPasteboard.generalPasteboard;
		NSString *content = [p stringForType:NSPasteboardTypeString];
		return (__bridge_retained CFTypeRef)content;
	}
}

CGFloat gio_viewHeight(CFTypeRef viewRef) {
	NSView *view = (__bridge NSView *)viewRef;
	return [view bounds].size.height;
}

CGFloat gio_viewWidth(CFTypeRef viewRef) {
	NSView *view = (__bridge NSView *)viewRef;
	return [view bounds].size.width;
}

CGFloat gio_getScreenBackingScale(void) {
	return [NSScreen.mainScreen backingScaleFactor];
}

CGFloat gio_getViewBackingScale(CFTypeRef viewRef) {
	NSView *view = (__bridge NSView *)viewRef;
	return [view.window backingScaleFactor];
}

void gio_hideCursor() {
	@autoreleasepool {
		[NSCursor hide];
	}
}

void gio_showCursor() {
	@autoreleasepool {
		[NSCursor unhide];
	}
}

void gio_setCursor(NSUInteger curID) {
	@autoreleasepool {
		switch (curID) {
			case 1:
				[NSCursor.arrowCursor set];
				break;
			case 2:
				[NSCursor.IBeamCursor set];
				break;
			case 3:
				[NSCursor.pointingHandCursor set];
				break;
			case 4:
				[NSCursor.crosshairCursor set];
				break;
			case 5:
				[NSCursor.resizeLeftRightCursor set];
				break;
			case 6:
				[NSCursor.resizeUpDownCursor set];
				break;
			default:
				[NSCursor.arrowCursor set];
				break;
		}
	}
}

static CVReturn displayLinkCallback(CVDisplayLinkRef dl, const CVTimeStamp *inNow, const CVTimeStamp *inOutputTime, CVOptionFlags flagsIn, CVOptionFlags *flagsOut, void *displayLinkContext) {
	gio_onFrameCallback(dl);
	return kCVReturnSuccess;
}

CFTypeRef gio_createDisplayLink(void) {
	CVDisplayLinkRef dl;
	CVDisplayLinkCreateWithActiveCGDisplays(&dl);
	CVDisplayLinkSetOutputCallback(dl, displayLinkCallback, nil);
	return dl;
}

int gio_startDisplayLink(CFTypeRef dl) {
	return CVDisplayLinkStart((CVDisplayLinkRef)dl);
}

int gio_stopDisplayLink(CFTypeRef dl) {
	return CVDisplayLinkStop((CVDisplayLinkRef)dl);
}

void gio_releaseDisplayLink(CFTypeRef dl) {
	CVDisplayLinkRelease((CVDisplayLinkRef)dl);
}

void gio_setDisplayLinkDisplay(CFTypeRef dl, uint64_t did) {
	CVDisplayLinkSetCurrentCGDisplay((CVDisplayLinkRef)dl, (CGDirectDisplayID)did);
}

NSPoint gio_cascadeTopLeftFromPoint(CFTypeRef windowRef, NSPoint topLeft) {
	NSWindow *window = (__bridge NSWindow *)windowRef;
	return [window cascadeTopLeftFromPoint:topLeft];
}

void gio_makeKeyAndOrderFront(CFTypeRef windowRef) {
	NSWindow *window = (__bridge NSWindow *)windowRef;
	[window makeKeyAndOrderFront:nil];
}

CFTypeRef gio_createWindow(CFTypeRef viewRef, const char *title, CGFloat width, CGFloat height, CGFloat minWidth, CGFloat minHeight, CGFloat maxWidth, CGFloat maxHeight) {
	@autoreleasepool {
		NSRect rect = NSMakeRect(0, 0, width, height);
		NSUInteger styleMask = NSTitledWindowMask |
			NSResizableWindowMask |
			NSMiniaturizableWindowMask |
			NSClosableWindowMask;

		NSWindow* window = [[NSWindow alloc] initWithContentRect:rect
													   styleMask:styleMask
														 backing:NSBackingStoreBuffered
														   defer:NO];
		if (minWidth > 0 || minHeight > 0) {
			window.contentMinSize = NSMakeSize(minWidth, minHeight);
		}
		if (maxWidth > 0 || maxHeight > 0) {
			window.contentMaxSize = NSMakeSize(maxWidth, maxHeight);
		}
		[window setAcceptsMouseMovedEvents:YES];
		window.title = [NSString stringWithUTF8String: title];
		NSView *view = (__bridge NSView *)viewRef;
		[window setContentView:view];
		[window makeFirstResponder:view];
		window.releasedWhenClosed = NO;
		window.delegate = globalWindowDel;
		return (__bridge_retained CFTypeRef)window;
	}
}

CFTypeRef gio_newMenu(const char *title) {
	@autoreleasepool {
		NSString *nsTitle = [NSString stringWithUTF8String:title];
		NSMenu *menu = [[NSMenu alloc] initWithTitle:nsTitle];
		return (__bridge_retained CFTypeRef)menu;
	}
}

void gio_menuAddItem(CFTypeRef menu, CFTypeRef menuItem) {
	@autoreleasepool {
		NSMenu *nsMenu = (__bridge NSMenu *)menu;
		NSMenuItem *nsMenuItem = (__bridge NSMenuItem *)menuItem;
		[nsMenu addItem:nsMenuItem];
	}
}

CFTypeRef gio_newSubMenu(CFTypeRef subMenu) {
	@autoreleasepool {
		NSMenu *nsSubMenu = (__bridge NSMenu *)subMenu;
		NSMenuItem *menu = [NSMenuItem new];
		[menu setSubmenu:nsSubMenu];
		return (__bridge_retained CFTypeRef)menu;
	}
}

CFTypeRef gio_mainMenu() {
	return (__bridge_retained CFTypeRef)[NSApp mainMenu];
}

CFTypeRef gio_newMenuItem(const char *title, const char *keyEquivalent, int modifiers, int tag) {
	@autoreleasepool {
		NSString *nsTitle = [NSString stringWithUTF8String:title];
		NSString *nsKeyEq = [NSString stringWithUTF8String:keyEquivalent];
		NSMenuItem *menuItem = [[NSMenuItem alloc] initWithTitle:nsTitle
														  action:@selector(applicationMenu:)
												   keyEquivalent:nsKeyEq];
		if (modifiers != 0) {
		  menuItem.keyEquivalentModifierMask = modifiers;
		}
		[menuItem setTag:tag];
		return (__bridge_retained CFTypeRef)menuItem;
	}
}

void gio_close(CFTypeRef windowRef) {
  NSWindow* window = (__bridge NSWindow *)windowRef;
  [window performClose:nil];
}

@implementation GioAppDelegate
- (void)applicationDidFinishLaunching:(NSNotification *)aNotification {
	[[NSRunningApplication currentApplication] activateWithOptions:(NSApplicationActivateAllWindows | NSApplicationActivateIgnoringOtherApps)];
	gio_onFinishLaunching();
}
- (void)applicationDidHide:(NSNotification *)aNotification {
	gio_onAppHide();
}
- (void)applicationWillUnhide:(NSNotification *)notification {
	gio_onAppShow();
}
- (void)applicationMenu:(id) sender {
    NSMenuItem * item = (NSMenuItem*)sender;
    int tag = [item tag];
    gio_onAppMenu(tag);
}
@end

void gio_main() {
	@autoreleasepool {
		[NSApplication sharedApplication];
		GioAppDelegate *del = [[GioAppDelegate alloc] init];
		[NSApp setDelegate:del];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];

		NSMenuItem *mainMenu = [NSMenuItem new];

		NSMenu *mainSubMenu = [NSMenu new];
		NSMenuItem *hideMenuItem = [[NSMenuItem alloc] initWithTitle:@"Hide"
															  action:@selector(hide:)
													   keyEquivalent:@"h"];
		[mainSubMenu addItem:hideMenuItem];
		NSMenuItem *quitMenuItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
															  action:@selector(terminate:)
													   keyEquivalent:@"q"];
		[mainSubMenu addItem:quitMenuItem];
		[mainMenu setSubmenu:mainSubMenu];
		NSMenu *menuBar = [NSMenu new];
		[menuBar addItem:mainMenu];
		[NSApp setMainMenu:menuBar];

		//NSMenuItem *newWindowItem = (__bridge NSMenuItem *)gio_newMenuItem("New Window", "n", 3);
		//NSMenu *fileSubMenu = (__bridge NSMenu *)gio_newMenu("File");
		//gio_menuAddItem((__bridge CFTypeRef)fileSubMenu, (__bridge CFTypeRef)newWindowItem);
		//NSMenuItem *fileMenu = (__bridge NSMenuItem *)gio_newSubMenu((__bridge CFTypeRef)fileSubMenu);
		//gio_menuAddItem(gio_mainMenu(), (__bridge CFTypeRef)fileMenu);

		//CFTypeRef newWindowItem = gio_newMenuItem("New Window", "n", 3);
		//CFTypeRef fileSubMenu = gio_newMenu("File");
		//gio_menuAddItem(fileSubMenu, newWindowItem);
		//CFTypeRef fileMenu = gio_newSubMenu(fileSubMenu);
		//gio_menuAddItem(gio_mainMenu(), fileMenu);

		globalWindowDel = [[GioWindowDelegate alloc] init];

		[NSApp run];
	}
}
