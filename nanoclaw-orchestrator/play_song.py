import webbrowser

# Search query
query = "maula mere maula"
# Construct YouTube search URL
url = f"https://www.youtube.com/results?search_query={query.replace(' ', '+')}"

# Open YouTube with the search query
webbrowser.open(url)

print(f"Opening YouTube to search for: {query}")
