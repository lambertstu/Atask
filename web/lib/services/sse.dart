import 'dart:async';
import 'dart:convert';
import 'dart:js_interop';
import 'package:web/web.dart' as web;
import '../models/event.dart';

class SseService {
  static const String baseUrl = 'http://127.0.0.1:8888';

  Stream<SessionEvent> connect(String sessionId) {
    final controller = StreamController<SessionEvent>();
    final url = '$baseUrl/api/sessions/$sessionId/events';
    final eventSource = web.EventSource(url);

    void handleMessage(web.Event event) {
      final msgEvent = event as web.MessageEvent;
      try {
        final data = jsonDecode(msgEvent.data.toString()) as Map<String, dynamic>;
        final sessionEvent = SessionEvent.fromJson(data);
        controller.add(sessionEvent);
      } catch (e) {
        controller.addError(e);
      }
    }

    void handleError(web.Event error) {
      controller.addError(error);
    }

    eventSource.addEventListener('message', handleMessage.toJS);
    eventSource.addEventListener('error', handleError.toJS);

    controller.onCancel = () {
      eventSource.close();
    };

    return controller.stream;
  }
}