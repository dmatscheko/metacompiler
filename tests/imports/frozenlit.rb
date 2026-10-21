# frozen_string_literal: true
# Required by ruby-test-features.rb: the magic comment above freezes the string
# literals of THIS file only, and a literal without interpolation is one shared
# object per text.
def fzl_lit; "fz"; end
def fzl_interp(n); "fz#{n}"; end
def fzl_words; %w[p q]; end
def fzl_heredoc
  <<~EOS
    hd
  EOS
end
def fzl_append
  "fz" << "x"
rescue FrozenError => e
  e.message
end
