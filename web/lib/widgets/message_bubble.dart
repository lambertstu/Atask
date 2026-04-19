import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import '../models/message.dart';
import 'tool_call_card.dart';

class MessageBubble extends StatefulWidget {
  final ChatMessage message;
  final Map<String, String>? toolResults;

  const MessageBubble({
    super.key,
    required this.message,
    this.toolResults,
  });

  @override
  State<MessageBubble> createState() => _MessageBubbleState();
}

class _MessageBubbleState extends State<MessageBubble> {
  bool _reasoningExpanded = false;
  bool _toolsExpanded = false;

  void _copyMessage(BuildContext context) {
    final content = widget.message.content ?? '';
    if (content.isEmpty) return;

    Clipboard.setData(ClipboardData(text: content));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('已复制'),
        duration: Duration(seconds: 1),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (widget.message.role == 'user') {
      return _buildUserBubble(context);
    } else if (widget.message.role == 'assistant') {
      return _buildAssistantBubble(context);
    }
    return const SizedBox.shrink();
  }

  Widget _buildUserBubble(BuildContext context) {
    final content = widget.message.content ?? '';
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          if (content.isNotEmpty)
            InkWell(
              onTap: () => _copyMessage(context),
              borderRadius: BorderRadius.circular(12),
              child: Padding(
                padding: const EdgeInsets.all(4),
                child: Icon(
                  Icons.copy,
                  size: 16,
                  color: Colors.grey[600],
                ),
              ),
            ),
          const SizedBox(width: 4),
          Flexible(
            child: Container(
              padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
              decoration: BoxDecoration(
                color: const Color(0xFF3B82F6),
                borderRadius: BorderRadius.circular(16),
              ),
              child: SelectableText(
                content,
                style: const TextStyle(color: Colors.white),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildAssistantBubble(BuildContext context) {
    final content = widget.message.content ?? '';
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: const Color(0xFF8B5CF6).withAlpha(30),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(
                color: const Color(0xFF8B5CF6).withAlpha(80),
                width: 1,
              ),
            ),
            child: const Icon(
              Icons.smart_toy,
              color: Color(0xFF8B5CF6),
              size: 20,
            ),
          ),
          const SizedBox(width: 8),
          Flexible(
            child: Container(
              padding: const EdgeInsets.only(top: 12, bottom: 12, left: 16, right: 16),
              decoration: BoxDecoration(
                color: Colors.grey[100],
                borderRadius: BorderRadius.circular(16),
              ),
              child: Stack(
                clipBehavior: Clip.none,
                children: [
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      if (content.isNotEmpty)
                        SelectionArea(
                          child: MarkdownBody(
                            data: content,
                            styleSheet: MarkdownStyleSheet(
                              p: TextStyle(color: Colors.grey[800]),
                              code: TextStyle(
                                backgroundColor: Colors.grey[300],
                                color: Colors.grey[800],
                              ),
                              codeblockPadding: const EdgeInsets.all(8),
                              codeblockDecoration: BoxDecoration(
                                color: Colors.grey[200],
                                borderRadius: BorderRadius.circular(8),
                              ),
                            ),
                          ),
                        ),
                      if (widget.message.reasoningContent != null &&
                          widget.message.reasoningContent!.isNotEmpty)
                        _buildReasoningSection(),
                      if (widget.message.toolCalls != null &&
                          widget.message.toolCalls!.isNotEmpty)
                        _buildToolCallsSection(),
                    ],
                  ),
                  if (content.isNotEmpty)
                    Positioned(
                      top: -4,
                      right: -4,
                      child: InkWell(
                        onTap: () => _copyMessage(context),
                        borderRadius: BorderRadius.circular(12),
                        child: Padding(
                          padding: const EdgeInsets.all(4),
                          child: Icon(
                            Icons.copy,
                            size: 16,
                            color: Colors.grey[600],
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildReasoningSection() {
    return Container(
      margin: const EdgeInsets.only(top: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          InkWell(
            onTap: () => setState(() => _reasoningExpanded = !_reasoningExpanded),
            child: Row(
              children: [
                Icon(
                  Icons.psychology,
                  size: 14,
                  color: Colors.grey[600],
                ),
                const SizedBox(width: 4),
                Text(
                  'Reasoning',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    color: Colors.grey[600],
                  ),
                ),
                const SizedBox(width: 4),
                Icon(
                  _reasoningExpanded ? Icons.expand_less : Icons.expand_more,
                  size: 14,
                  color: Colors.grey[600],
                ),
              ],
            ),
          ),
          if (_reasoningExpanded)
            Container(
              margin: const EdgeInsets.only(top: 6),
              padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
              decoration: BoxDecoration(
                color: Colors.grey[200],
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                widget.message.reasoningContent!,
                style: TextStyle(
                  fontSize: 12,
                  color: Colors.grey[700],
                ),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildToolCallsSection() {
    return Container(
      margin: const EdgeInsets.only(top: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          InkWell(
            onTap: () => setState(() => _toolsExpanded = !_toolsExpanded),
            child: Row(
              children: [
                Icon(
                  Icons.build,
                  size: 14,
                  color: Colors.grey[600],
                ),
                const SizedBox(width: 4),
                Text(
                  'Tool Calls (${widget.message.toolCalls!.length})',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    color: Colors.grey[600],
                  ),
                ),
                const SizedBox(width: 4),
                Icon(
                  _toolsExpanded ? Icons.expand_less : Icons.expand_more,
                  size: 14,
                  color: Colors.grey[600],
                ),
              ],
            ),
          ),
          if (_toolsExpanded)
            ...widget.message.toolCalls!.map((tc) => ToolCallCard(
              toolCall: tc,
              result: widget.toolResults?[tc.id],
            )),
        ],
      ),
    );
  }
}