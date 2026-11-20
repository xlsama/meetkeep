import AVFoundation
import CoreAudio
import Foundation

func makeRecorder() throws -> any Recorder { NativeRecorder() }

final class NativeRecorder: Recorder {
    func frames(source: Source) -> AsyncThrowingStream<Data, Error> {
        AsyncThrowingStream { continuation in
            let task = Task {
                do {
                    let sink = FrameSink { continuation.yield($0) }
                    let stop: () -> Void
                    switch source {
                    case .microphone:
                        guard await Microphone.allowed() else { throw CaptureError.denied }
                        stop = try Microphone.start(sink)
                    case .system:
                        stop = try SystemAudio.start(sink)
                    }
                    await withTaskCancellationHandler {
                        while !Task.isCancelled {
                            try? await Task.sleep(nanoseconds: 3_600_000_000_000)
                        }
                    } onCancel: {
                        stop()
                    }
                } catch {
                    continuation.finish(throwing: error)
                }
            }
            continuation.onTermination = { _ in task.cancel() }
        }
    }
}

final class FrameSink {
    private static let frameBytes = 3200
    private let emit: (Data) -> Void
    private let output = AVAudioFormat(
        commonFormat: .pcmFormatInt16, sampleRate: 16000, channels: 1, interleaved: true)!
    private var input: AVAudioFormat?
    private var converter: AVAudioConverter?
    private var pending = Data()

    init(emit: @escaping (Data) -> Void) { self.emit = emit }

    func push(_ buffer: AVAudioPCMBuffer, channels: Int) {
        guard let source = buffer.floatChannelData else { return }
        let frames = Int(buffer.frameLength)
        let available = Int(buffer.format.channelCount)
        let mixed = max(1, min(channels, available))
        guard frames > 0 else { return }
        let rate = buffer.format.sampleRate
        if input?.sampleRate != rate {
            let format = AVAudioFormat(
                commonFormat: .pcmFormatFloat32, sampleRate: rate, channels: 1, interleaved: false)!
            input = format
            converter = AVAudioConverter(from: format, to: output)
        }
        guard let input, let converter,
            let mono = AVAudioPCMBuffer(pcmFormat: input, frameCapacity: AVAudioFrameCount(frames)),
            let target = mono.floatChannelData?[0]
        else { return }
        mono.frameLength = AVAudioFrameCount(frames)
        let scale = 1 / Float(mixed)
        if buffer.format.isInterleaved {
            let base = source[0]
            for i in 0..<frames {
                var sum: Float = 0
                for c in 0..<mixed { sum += base[i * available + c] }
                target[i] = sum * scale
            }
        } else {
            for i in 0..<frames {
                var sum: Float = 0
                for c in 0..<mixed { sum += source[c][i] }
                target[i] = sum * scale
            }
        }
        let capacity = AVAudioFrameCount(Double(frames) * 16000 / rate) + 64
        guard let out = AVAudioPCMBuffer(pcmFormat: output, frameCapacity: capacity) else { return }
        var fed = false
        var failure: NSError?
        converter.convert(to: out, error: &failure) { _, status in
            if fed {
                status.pointee = .noDataNow
                return nil
            }
            fed = true
            status.pointee = .haveData
            return mono
        }
        guard failure == nil, out.frameLength > 0, let samples = out.int16ChannelData?[0] else { return }
        pending.append(
            UnsafeBufferPointer(start: samples, count: Int(out.frameLength)))
        while pending.count >= Self.frameBytes {
            emit(pending.prefix(Self.frameBytes))
            pending.removeFirst(Self.frameBytes)
        }
    }
}

enum Microphone {
    static func allowed() async -> Bool {
        switch AVCaptureDevice.authorizationStatus(for: .audio) {
        case .authorized: return true
        case .notDetermined: return await AVCaptureDevice.requestAccess(for: .audio)
        default: return false
        }
    }

    static func start(_ sink: FrameSink) throws -> () -> Void {
        let engine = AVAudioEngine()
        let node = engine.inputNode
        try? node.setVoiceProcessingEnabled(true)
        if node.isVoiceProcessingEnabled {
            node.voiceProcessingOtherAudioDuckingConfiguration =
                AVAudioVoiceProcessingOtherAudioDuckingConfiguration(
                    enableAdvancedDucking: false, duckingLevel: .min)
            node.isVoiceProcessingAGCEnabled = false
        }
        let format = node.outputFormat(forBus: 0)
        guard format.sampleRate > 0, format.channelCount > 0 else { throw NativeError.unsupported }
        let channels = node.isVoiceProcessingEnabled ? 1 : Int(format.channelCount)
        node.installTap(onBus: 0, bufferSize: 4096, format: format) { buffer, _ in
            sink.push(buffer, channels: channels)
        }
        engine.prepare()
        try engine.start()
        return {
            node.removeTap(onBus: 0)
            if node.isVoiceProcessingEnabled { try? node.setVoiceProcessingEnabled(false) }
            engine.stop()
        }
    }
}

enum SystemAudio {
    static func start(_ sink: FrameSink) throws -> () -> Void {
        let description = CATapDescription(stereoGlobalTapButExcludeProcesses: ownProcess())
        description.name = "meetkeep-tap"
        description.isPrivate = true
        description.muteBehavior = .unmuted

        var tap = AudioObjectID(kAudioObjectUnknown)
        try check(AudioHardwareCreateProcessTap(description, &tap), "create the tap")

        var aggregate = AudioObjectID(kAudioObjectUnknown)
        var proc: AudioDeviceIOProcID?
        let teardown = {
            if aggregate != kAudioObjectUnknown {
                if let proc {
                    AudioDeviceStop(aggregate, proc)
                    AudioDeviceDestroyIOProcID(aggregate, proc)
                }
                AudioHardwareDestroyAggregateDevice(aggregate)
            }
            AudioHardwareDestroyProcessTap(tap)
        }

        do {
            var address = AudioObjectPropertyAddress(
                mSelector: kAudioTapPropertyFormat,
                mScope: kAudioObjectPropertyScopeGlobal,
                mElement: kAudioObjectPropertyElementMain)
            var stream = AudioStreamBasicDescription()
            var size = UInt32(MemoryLayout<AudioStreamBasicDescription>.size)
            try check(
                AudioObjectGetPropertyData(tap, &address, 0, nil, &size, &stream), "read the tap's format")
            guard let format = AVAudioFormat(streamDescription: &stream) else {
                throw NativeError.unsupported
            }

            let device: [String: Any] = [
                kAudioAggregateDeviceNameKey: "meetkeep-aggregate",
                kAudioAggregateDeviceUIDKey: UUID().uuidString,
                kAudioAggregateDeviceIsPrivateKey: true,
                kAudioAggregateDeviceTapAutoStartKey: true,
                kAudioAggregateDeviceTapListKey: [[kAudioSubTapUIDKey: description.uuid.uuidString]],
            ]
            try check(
                AudioHardwareCreateAggregateDevice(device as CFDictionary, &aggregate),
                "create the aggregate device")

            let queue = DispatchQueue(label: "meetkeep.capture.system")
            let channels = Int(format.channelCount)
            try check(
                AudioDeviceCreateIOProcIDWithBlock(&proc, aggregate, queue) { _, input, _, _, _ in
                    guard
                        let buffer = AVAudioPCMBuffer(
                            pcmFormat: format, bufferListNoCopy: input, deallocator: nil)
                    else { return }
                    sink.push(buffer, channels: channels)
                }, "create the IO proc")
            try check(AudioDeviceStart(aggregate, proc), "start the device")
        } catch {
            teardown()
            throw error
        }
        return teardown
    }

    private static func ownProcess() -> [AudioObjectID] {
        var address = AudioObjectPropertyAddress(
            mSelector: kAudioHardwarePropertyTranslatePIDToProcessObject,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain)
        var pid = getpid()
        var object = AudioObjectID(kAudioObjectUnknown)
        var size = UInt32(MemoryLayout<AudioObjectID>.size)
        let status = AudioObjectGetPropertyData(
            AudioObjectID(kAudioObjectSystemObject), &address,
            UInt32(MemoryLayout<pid_t>.size), &pid, &size, &object)
        return status == noErr && object != kAudioObjectUnknown ? [object] : []
    }

    private static func check(_ status: OSStatus, _ action: String) throws {
        guard status == noErr else {
            throw NSError(
                domain: NSOSStatusErrorDomain, code: Int(status),
                userInfo: [NSLocalizedDescriptionKey: "Could not \(action) (\(status))"])
        }
    }
}
