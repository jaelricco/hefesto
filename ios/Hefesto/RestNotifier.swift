import Foundation
import UserNotifications

/// A local notification when the planned rest ends, so the timer reaches the
/// athlete with the phone locked. One at a time: a new rest replaces it.
enum RestNotifier {
    private static let id = "fit.hefesto.rest"

    static func schedule(at end: Date) {
        let interval = end.timeIntervalSinceNow
        guard interval > 1 else { return }
        Task {
            let center = UNUserNotificationCenter.current()
            guard (try? await center.requestAuthorization(options: [.alert, .sound])) == true else { return }
            let content = UNMutableNotificationContent()
            content.title = String(localized: "Rest done")
            content.body = String(localized: "Ready for the next set when you are.")
            content.sound = .default
            let trigger = UNTimeIntervalNotificationTrigger(timeInterval: interval, repeats: false)
            center.removePendingNotificationRequests(withIdentifiers: [id])
            try? await center.add(UNNotificationRequest(identifier: id, content: content, trigger: trigger))
        }
    }

    static func cancel() {
        UNUserNotificationCenter.current().removePendingNotificationRequests(withIdentifiers: [id])
    }
}
