import 'dart:async';
import 'dart:convert';
import 'dart:js_interop';
import 'package:web/web.dart' as web;
import '../models/event.dart';

class SseService {
  static const String baseUrl = 'http://127.0.0.1:8888';
  
  web.EventSource? _eventSource;
  StreamController<SessionEvent>? _controller;
  
  Stream<SessionEvent> connect(String sessionId) {
    _controller = StreamController<SessionEvent>();
    final url = '$baseUrl/api/sessions/$sessionId/events';
    
    _eventSource = web.EventSource(url);
    
    void handleMessage(web.Event event) {
      final msgEvent = event as web.MessageEvent;
      try {
        final data = jsonDecode(msgEvent.data.toString()) as Map<String, dynamic>;
        final sessionEvent = SessionEvent.fromJson(data);
        _controller!.add(sessionEvent);
      } catch (e) {
        _controller!.addError(e);
      }
    }
    
    void handleError(web.Event error) {
      _controller!.addError(error);
    }
    
    _eventSource!.addEventListener('message', handleMessage.toJS);
    _eventSource!.addEventListener('error', handleError.toJS);
    
    return _controller!.stream;
  }
  
  void disconnect() {
    _eventSource?.close();
    _controller?.close();
    _eventSource = null;
    _controller = null;
  }
}