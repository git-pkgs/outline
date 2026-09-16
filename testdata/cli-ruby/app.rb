require_relative "lib/helper"

class Worker
  def run(path)
    File.read(path)
  end
end
