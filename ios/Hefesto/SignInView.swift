import AuthenticationServices
import CryptoKit
import HefestoAuth
import SwiftUI

struct SignInView: View {
    @Environment(AppModel.self) private var model
    @State private var isRegistering = false
    @State private var email = ""
    @State private var password = ""
    @State private var displayName = ""
    @State private var working = false
    @State private var message: LocalizedStringKey?
    @State private var detail: String?
    @State private var nonce = ""

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Email", text: $email)
                        .textContentType(.username)
                        .keyboardType(.emailAddress)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    SecureField("Password", text: $password)
                        .textContentType(isRegistering ? .newPassword : .password)
                    if isRegistering {
                        TextField("Name (optional)", text: $displayName)
                            .textContentType(.name)
                    }
                } footer: {
                    if isRegistering { Text("At least 10 characters.") }
                }

                Section {
                    Button {
                        Task { await submit() }
                    } label: {
                        Text(isRegistering ? LocalizedStringKey("Create account") : "Sign in")
                            .frame(maxWidth: .infinity, minHeight: 44)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(working || email.isEmpty || password.isEmpty)

                    SignInWithAppleButton(isRegistering ? .signUp : .signIn) { request in
                        nonce = Self.randomNonce()
                        request.requestedScopes = [.fullName]
                        request.nonce = Self.sha256(nonce)
                    } onCompletion: { result in
                        Task { await apple(result) }
                    }
                    .signInWithAppleButtonStyle(.white)
                    .frame(minHeight: 44)
                    .disabled(working)
                }
                .listRowInsets(EdgeInsets())
                .listRowBackground(Color.clear)

                Section {
                    Button(isRegistering ? LocalizedStringKey("I already have an account") : "Create an account") {
                        isRegistering.toggle()
                        message = nil
                    }
                }

                if let message {
                    Section {
                        Text(message).foregroundStyle(.red)
                        if let detail { Text(verbatim: detail).font(.footnote).foregroundStyle(.secondary) }
                    }
                }
            }
            .navigationTitle("Hefesto")
        }
    }

    private func submit() async {
        working = true
        defer { working = false }
        do {
            if isRegistering {
                try await model.auth.register(
                    email: email, password: password, displayName: displayName.isEmpty ? nil : displayName,
                    timezone: TimeZone.current.identifier, locale: Locale.current.identifier(.bcp47))
            } else {
                try await model.auth.login(email: email, password: password)
            }
            password = ""
            await model.signedIn()
        } catch {
            show(error)
        }
    }

    private func apple(_ result: Result<ASAuthorization, any Error>) async {
        guard case let .success(authorization) = result,
              let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
              let tokenData = credential.identityToken,
              let token = String(data: tokenData, encoding: .utf8)
        else {
            if case let .failure(error) = result, (error as? ASAuthorizationError)?.code == .canceled { return }
            message = "Sign in with Apple did not complete."
            return
        }
        let name = credential.fullName.flatMap { PersonNameComponentsFormatter().string(from: $0) }
        working = true
        defer { working = false }
        do {
            try await model.auth.signInWithApple(
                identityToken: token, rawNonce: nonce, displayName: name?.isEmpty == false ? name : nil)
            await model.signedIn()
        } catch {
            show(error)
        }
    }

    private func show(_ error: any Error) {
        detail = nil
        switch error as? AuthError {
        case .invalidCredentials: message = "Email or password is wrong."
        case .emailTaken: message = "An account with this email exists. Sign in instead."
        case let .invalid(why): message = "Please check your details."; detail = why
        case .rateLimited: message = "Too many attempts. Try again in a few minutes."
        case .forbidden: message = "This account cannot sign in."
        default: message = "Could not reach Hefesto. Check your connection."
        }
    }

    static func randomNonce() -> String {
        var bytes = [UInt8](repeating: 0, count: 32)
        for i in bytes.indices { bytes[i] = UInt8.random(in: .min ... .max) }
        return Data(bytes).base64EncodedString()
    }

    static func sha256(_ s: String) -> String {
        SHA256.hash(data: Data(s.utf8)).map { String(format: "%02x", $0) }.joined()
    }
}
